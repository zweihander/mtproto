package mtproto

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/service"
	"github.com/xelaj/mtproto/utils"
)

type MTProto struct {
	conn         net.Conn
	stopRoutines context.CancelFunc // остановить ping, read, и подобные горутины

	creds *SessionCredentials

	sessionID int64

	pending *requestStore
	acks    *ackStore

	// идентификаторы сообщений, нужны что бы посылать и принимать сообщения.
	seqNo int32
	msgId int64

	// не знаю что это но как-то используется
	lastSeqNo int32

	// пока непонятно для чего, кажется это нужно клиенту конкретно телеграма
	dclist map[int32]string

	sessionStore          SessionStore
	serverRequestHandlers []func(i interface{}) bool
}

func NewMTProto(host string, publicKey *rsa.PublicKey, sess SessionStore) (*MTProto, error) {
	conn, err := mtDial(host)
	if err != nil {
		return nil, err
	}

	if sess == nil {
		sess = &noOpSessionStore{}
	}

	creds, err := sess.Get()
	if err != nil {
		if !errors.Is(err, ErrNoCredentials) {
			return nil, fmt.Errorf("get credentials from store: %w", err)
		}

		fmt.Println("store are empty, create new creds")
		creds, err = handshake(conn, publicKey)
		if err != nil {
			return nil, err
		}

		if err := sess.Set(creds); err != nil {
			return nil, err
		}
	} else {
		fmt.Println("using existing creds from store")
	}

	m := &MTProto{
		conn:         conn,
		sessionID:    utils.GenerateSessionID(),
		creds:        creds,
		pending:      newRequestStore(),
		acks:         newAckStore(),
		sessionStore: sess,
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.stopRoutines = cancel

	go m.startReadingResponses(ctx)
	go m.startPinging(ctx)
	return m, nil
}

func mtDial(host string) (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	// https://core.telegram.org/mtproto/mtproto-transports#intermediate
	_, err = conn.Write([]byte{0xee, 0xee, 0xee, 0xee})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (m *MTProto) MakeRequest(req tl.Object, resp interface{}) error {
	err := m.sendPacket(req, resp)
	// если пришел ответ типа badServerSalt, то отправляем данные заново
	if errors.As(err, &service.ErrorSessionConfigsChanged{}) {
		return m.MakeRequest(req, resp)
	}

	return err
}

func (m *MTProto) Close() error {
	// stop all routines
	m.stopRoutines()

	// TODO: закрыть каналы
	return m.conn.Close()
}

// startPinging пингует сервер что все хорошо, клиент в сети
// нужно просто запустить
func (m *MTProto) startPinging(ctx context.Context) {
	ticker := time.Tick(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			_, err := m.Ping(0xCADACADA)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *MTProto) startReadingResponses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := m.readFromConn(ctx)
			if err != nil {
				panic(err)
			}

			response, err := m.decodeRecievedData(data)
			if err != nil {
				panic(err)
			}

			// NOTE:
			// Зачем сюда передавать m.msgId, m.seqNo
			// если у сообщения есть методы GetMsgID() и GetSeqNo()?
			if err := m.processResponse(
				atomic.LoadInt64(&m.msgId),
				atomic.LoadInt32(&m.seqNo),
				response.Msg,
			); err != nil {
				panic(err)
			}
		}
	}
}

func (m *MTProto) processResponse(msgID int64, seqNo int32, data []byte) error {
	object, err := tl.DecodeRegistered(data)
	if err != nil {
		return fmt.Errorf("decode base message: %w", err)
	}

	switch message := object.(type) {
	case *service.RpcResult:

		req, found := m.pending.Pop(message.ReqMsgID)
		if !found {
			fmt.Printf("pending request for messageID %d not found\n", message.ReqMsgID)
			break
		}

		rpcMessageObject, err := tl.DecodeRegistered(message.Payload)
		if err != nil {
			// если не смогли заанмаршалить в зареганный тип
			// пробуем анмаршалить в тип прокинутый юзером
			//
			// Такое случается потому что DecodeRegistered (в отличие от Decode) не умеет
			// анмаршалить CrcVector, но можно его научить
			req.echan <- tl.Decode(message.Payload, req.response)
			break
		}

		// джедайские трюки
		switch rpcMessage := rpcMessageObject.(type) {
		case *service.GzipPacked:
			req.echan <- tl.Decode(rpcMessage.PackedData, req.response)
		case *service.RpcError:
			req.echan <- rpcMessage
		default: // если в rpc хз что, то анмаршалим его пейлоад в тот тип который запросил юзер

			// NOTE:
			// мб сделать свитч чисто по CRC чтобы убрать повторный анмаршал?
			// или установить значение rpcMessageObject в req.response через reflect?
			req.echan <- tl.Decode(message.Payload, req.response)
		}

	case *service.MessageContainer:
		println("MessageContainer")
		for _, v := range *message {
			// NOTE:
			// Зачем сюда передавать m.msgId, m.seqNo
			// если у сообщения есть методы GetMsgID() и GetSeqNo()?
			err := m.processResponse(v.MsgID, v.SeqNo, v.Msg)
			if err != nil {
				return errors.Wrap(err, "processing item in container")
			}
		}

	case *service.BadServerSalt:
		atomic.StoreInt64(&m.creds.ServerSalt, message.NewSalt)
		m.sessionStore.Set(m.creds)
		m.pending.ForEach(func(_ int64, req pendingRequest) {
			req.echan <- &service.ErrorSessionConfigsChanged{}
		})

	case *service.NewSessionCreated:
		println("session created")
		atomic.StoreInt64(&m.creds.ServerSalt, message.ServerSalt)
		m.sessionStore.Set(m.creds)

	case *service.Pong:
		// игнорим, пришло и пришло, че бубнить то

	case *service.MsgsAck:
		m.acks.GotMultipleAcks(message.MsgIds)

	case *service.BadMsgNotification:
		req, found := m.pending.Pop(message.BadMsgID)
		if !found {
			fmt.Printf("pending request for messageID %d not found\n", message.BadMsgID)
			break
		}

		req.echan <- message
	default:
		panic(fmt.Sprintf("type %T not handled", message))
	}

	if (seqNo & 1) != 0 {
		// NOTE:
		// похоже MsgsAck можно кидать Ack на несколько сообщений сразу
		// Мб отправлять их батчами для меньшего жора сети?
		err = m.MakeRequest(&service.MsgsAck{MsgIds: []int64{int64(msgID)}}, nil)
		if err != nil {
			return errors.Wrap(err, "sending ack")
		}
	}

	return nil
}
