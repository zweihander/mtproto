package mtproto

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"sync"
	"time"

	bus "github.com/asaskevich/EventBus"
	"github.com/pkg/errors"
	"github.com/xelaj/errs"
	"github.com/xelaj/go-dry"

	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/serialize"
	"github.com/xelaj/mtproto/utils"
)

type MTProto struct {
	addr         string
	conn         *net.TCPConn
	stopRoutines context.CancelFunc // остановить ping, read, и подобные горутины

	// ключ авторизации. изменять можно только через setAuthKey
	authKey []byte

	// хеш ключа авторизации. изменять можно только через setAuthKey
	authKeyHash []byte

	// соль сессии
	serverSalt int64
	encrypted  bool
	sessionId  int64

	// общий мьютекс
	mutex         *sync.Mutex
	pending       map[int64]pendingRequest
	idsToAck      map[int64]struct{}
	idsToAckMutex sync.Mutex

	// идентификаторы сообщений, нужны что бы посылать и принимать сообщения.
	seqNo int32
	msgId int64

	// не знаю что это но как-то используется
	lastSeqNo int32

	// пока непонятно для чего, кажется это нужно клиенту конкретно телеграма
	dclist map[int32]string

	// шина сообщений, используется для разных нотификаций, описанных в константах нотификации
	bus bus.Bus

	// путь до файла токена сессии.
	tokensStorage string

	// один из публичных ключей telegram. нужен только для создания сессии.
	publicKey *rsa.PublicKey

	// serviceChannel нужен только на время создания ключей, т.к. это
	// не RpcResult, поэтому все данные отдаются в один поток без
	// привязки к MsgID
	serviceChannel       chan []byte
	serviceModeActivated bool

	serverRequestHandlers []customHandlerFunc
}

type customHandlerFunc = func(i interface{}) bool

type Config struct {
	AuthKeyFile string
	ServerHost  string
	PublicKey   *rsa.PublicKey
}

func NewMTProto(c Config) (*MTProto, error) {
	m := new(MTProto)
	m.tokensStorage = c.AuthKeyFile

	err := m.LoadSession()
	if err == nil {
		m.encrypted = true
	} else if errs.IsNotFound(err) {
		m.addr = c.ServerHost
		m.encrypted = false
	} else {
		return nil, errors.Wrap(err, "loading session")
	}

	m.sessionId = utils.GenerateSessionID()
	m.serviceChannel = make(chan []byte)
	m.publicKey = c.PublicKey
	m.serverRequestHandlers = make([]customHandlerFunc, 0)
	m.resetAck()

	return m, nil
}

func (m *MTProto) CreateConnection() error {
	// connect
	tcpAddr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "resolving tcp")
	}
	m.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return errors.Wrap(err, "dialing tcp")
	}

	// https://core.telegram.org/mtproto/mtproto-transports#intermediate
	_, err = m.conn.Write([]byte{0xee, 0xee, 0xee, 0xee})
	if err != nil {
		return errors.Wrap(err, "writing first byte")
	}

	ctx, cancelfunc := context.WithCancel(context.Background())
	m.stopRoutines = cancelfunc

	// start reading responses from the server
	go m.startReadingResponses(ctx)

	// get new authKey if need
	if !m.encrypted {
		println("not encrypted, creating auth key")
		err = m.makeAuthKey()
		fmt.Println("authkey status:", err)
		if err != nil {
			return errors.Wrap(err, "making auth key")
		}
	}

	// start goroutines
	m.mutex = &sync.Mutex{}
	m.pending = make(map[int64]pendingRequest)
	// start keepalive pinging
	go m.startPinging(ctx)

	return nil
}

func (m *MTProto) makeRequest2(req tl.Object, resp interface{}) error {
	err := m.sendPacket2(req, resp)
	// если пришел ответ типа badServerSalt, то отправляем данные заново
	if errors.Is(err, &serialize.ErrorSessionConfigsChanged{}) {
		return m.makeRequest2(req, resp)
	}

	return err
}

func (m *MTProto) Disconnect() error {
	// stop all routines
	m.stopRoutines()

	err := m.conn.Close()
	if err != nil {
		return errors.Wrap(err, "closing TCP connection")
	}

	// TODO: закрыть каналы

	// возвращаем в false, потому что мы теряем конфигурацию
	// сессии, и можем ее потерять во время отключения.
	m.encrypted = false

	return nil
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

			if m.serviceModeActivated {
				fmt.Println("got service message")
				m.serviceChannel <- response.GetMsg()
				fmt.Println("service message pushed")
				continue
			}

			err = m.processResponse(int(m.msgId), int(m.seqNo), response.GetMsg())
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *MTProto) processResponse(msgID, seqNo int, data []byte) error {
	object, err := tl.DecodeRegistered(data)
	if err != nil {
		return fmt.Errorf("decode base message: %w", err)
	}

	switch message := object.(type) {
	case *serialize.RpcResult:
		m.mutex.Lock()
		req, found := m.pending[message.ReqMsgID]
		if !found {
			m.mutex.Unlock()
			fmt.Printf("pending request for message %d not found\n", message.ReqMsgID)
			return nil
		}
		delete(m.pending, message.ReqMsgID)
		m.mutex.Unlock()

		ob, err := tl.DecodeRegistered(message.Payload)
		if err != nil {
			req.echan <- fmt.Errorf("decode rpc: %w", err)
			return nil
		}

		switch obj := ob.(type) {
		case *serialize.GzipPacked:
			// джедайские трюки
			if req.response != nil {
				req.echan <- tl.Decode(obj.Payload, req.response)
				return nil
			}

			ob, err = tl.DecodeRegistered(obj.Payload)
			req.response = ob
			req.echan <- err
			return nil
		default:
			panic(fmt.Sprintf("type %T not handled", obj))
		}

		req.echan <- tl.Decode(message.Payload, req.response)
	case *serialize.MessageContainer:
		println("MessageContainer")
		for _, v := range *message {
			err := m.processResponse(int(v.MsgID), int(v.SeqNo), v.GetMsg())
			if err != nil {
				return errors.Wrap(err, "processing item in container")
			}
		}

	case *serialize.BadServerSalt:
		m.serverSalt = message.NewSalt
		err := m.SaveSession()
		dry.PanicIfErr(err)

		m.mutex.Lock()
		// TODO: check id
		for _, v := range m.pending {
			v.echan <- &serialize.ErrorSessionConfigsChanged{}
		}
		m.mutex.Unlock()

	case *serialize.NewSessionCreated:
		println("session created")
		m.serverSalt = message.ServerSalt
		err := m.SaveSession()
		if err != nil {
			panic(err)
		}

	case *serialize.Pong:
		// игнорим, пришло и пришло, че бубнить то

	case *serialize.MsgsAck:
		for _, id := range message.MsgIds {
			m.gotAck(id)
		}

	case *serialize.BadMsgNotification:
		panic(message)
		return BadMsgErrorFromNative(message)

		// case *serialize.RpcResult:
		// 	obj := message.Obj
		// 	if v, ok := obj.(*serialize.GzipPacked); ok {
		// 		obj = v.Obj
		// 	}

		// err := m.writeRPCResponse(int(message.ReqMsgID), obj)
		// if err != nil {
		// 	return errors.Wrap(err, "writing RPC response")
		// }

	default:
		processed := false
		for _, f := range m.serverRequestHandlers {
			processed = f(message)
			if processed {
				break
			}
		}
		if !processed {
			panic(fmt.Errorf("got nonsystem message from server: %T", message))
		}
	}

	if (seqNo & 1) != 0 {
		err = m.MakeRequest2(&serialize.MsgsAck{MsgIds: []int64{int64(msgID)}}, nil)
		if err != nil {
			return errors.Wrap(err, "sending ack")
		}
	}

	return nil
}
