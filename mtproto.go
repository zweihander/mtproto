package mtproto

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
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
	mutex *sync.Mutex

	// msgsIdDecodeAsVector показывает, что определенный ответ сервера нужно декодировать как
	// слайс. Это костыль, т.к. MTProto ЧЕТКО указывает, что ответы это всегда объекты, но
	// вектор (слайс) это как бы тоже объект. Из-за этого приходится четко указывать, что
	// сообщения с определенным msgID нужно декодировать как слайс, а не объект
	msgsIdDecodeAsVector map[int64]reflect.Type
	msgsIdToResp         map[int64]chan tl.Object
	idsToAck             map[int64]struct{}
	idsToAckMutex        sync.Mutex

	// каналы, которые ожидают ответа rpc. ответ записывается в канал и удаляется
	responseChannels map[int64]chan tl.Object

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
	serviceChannel       chan tl.Object
	serviceModeActivated bool

	//! DEPRECATED RecoverFunc используется только до того момента, когда из пакета будут убраны все паники
	RecoverFunc func(i interface{})
	// если задан, то в канал пишутся ошибки
	Warnings chan error

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
	m.serviceChannel = make(chan tl.Object)
	m.publicKey = c.PublicKey
	m.responseChannels = make(map[int64]chan tl.Object)
	m.msgsIdDecodeAsVector = make(map[int64]reflect.Type)
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
	m.startReadingResponses(ctx)

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
	m.msgsIdToResp = make(map[int64]chan tl.Object)
	m.mutex = &sync.Mutex{}

	// start keepalive pinging
	m.startPinging(ctx)

	go func() {
		for {
			warn := <-m.Warnings
			panic(warn)
		}
	}()

	return nil
}

// отправить запрос
func (m *MTProto) makeRequest(data tl.Object, as reflect.Type) (tl.Object, error) {
	resp, err := m.sendPacketNew(data, as)
	if err != nil {
		return nil, errors.Wrap(err, "sending message")
	}
	response := <-resp

	if _, ok := response.(*serialize.ErrorSessionConfigsChanged); ok {
		// если пришел ответ типа badServerSalt, то отправляем данные заново
		return m.makeRequest(data, as)
	}
	if e, ok := response.(*serialize.RpcError); ok {
		return nil, RpcErrorToNative(e)
	}

	return response, nil
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
	go func() {
		defer m.recoverGoroutine()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				_, err := m.Ping(0xCADACADA)
				if err != nil {
					if m.Warnings != nil {
						m.Warnings <- errors.Wrap(err, "ping unsuccsesful")
					}
				}
			}
		}
	}()
}

func (m *MTProto) startReadingResponses(ctx context.Context) {
	go func() {
		defer m.recoverGoroutine()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, err := m.readFromConn(ctx)
				if err != nil {
					m.Warnings <- errors.Wrap(err, "reading from connection")
					continue
				}

				response, err := m.decodeRecievedData(data)
				if err != nil {
					m.Warnings <- errors.Wrap(err, "decoding received data")
					continue
				}

				if m.serviceModeActivated {
					fmt.Println("servmode")
					// сервисные сообщения ГАРАНТИРОВАННО в теле содержат TL.
					obj, err := tl.DecodeRegistered(response.GetMsg())
					if err != nil {
						m.Warnings <- err
						continue
					}

					m.serviceChannel <- obj
				} else {
					err = m.processResponse(int(m.msgId), int(m.seqNo), response)
					if err != nil {
						m.Warnings <- errors.Wrap(err, "processing response")
					}
				}
			}
		}
	}()
}

// TODO: msgId, seqNo идентичны тем что в msg???
func (m *MTProto) processResponse(msgId, seqNo int, msg serialize.CommonMessage) error {
	// сначала декодируем исключения

	// TODO: может как-то поопрятней сделать? а то очень кринжово, функция занимается не тем, чем должна
	var data tl.Object
	// если это ответ Rpc, то там может быть слайс вместо объекта, надо проверить указывали ли мы,
	// что ответ с этим MsgId нужно декодировать как слайс, а не объект
	if binary.LittleEndian.Uint32(msg.GetMsg()[:tl.WordLen]) == serialize.CrcRpcResult {
		r := tl.NewReadCursor(bytes.NewBuffer(msg.GetMsg()))
		if _, err := r.PopCRC(); err != nil {
			return err
		}

		rpc := &serialize.RpcResult{}
		msgID := binary.LittleEndian.Uint64(msg.GetMsg()[tl.WordLen : tl.WordLen+tl.LongLen])
		if typ, ok := m.msgsIdDecodeAsVector[int64(msgID)]; ok {
			delete(m.msgsIdDecodeAsVector, int64(msgID))

			if err := rpc.DecodeFromButItsVector(r, typ); err != nil {
				return err
			}
		} else {
			rest, err := r.GetRestOfMessage()
			if err != nil {
				return err
			}

			if err := tl.Decode(rest, rpc); err != nil {
				return err
			}
		}
		data = rpc
	} else {
		d, err := tl.DecodeRegistered(msg.GetMsg())
		if err != nil {
			return err
		}

		data = d
	}

	switch message := data.(type) {
	case *serialize.MessageContainer:
		println("MessageContainer")
		for _, v := range *message {
			err := m.processResponse(int(v.MsgID), int(v.SeqNo), v)
			if err != nil {
				return errors.Wrap(err, "processing item in container")
			}
		}

	case *serialize.BadServerSalt:
		m.serverSalt = message.NewSalt
		err := m.SaveSession()
		dry.PanicIfErr(err)

		m.mutex.Lock()
		for _, v := range m.responseChannels {
			v <- &serialize.ErrorSessionConfigsChanged{}
		}
		m.mutex.Unlock()

	case *serialize.NewSessionCreated:
		println("session created")
		m.serverSalt = message.ServerSalt
		err := m.SaveSession()
		if err != nil {
			if m.Warnings != nil {
				m.Warnings <- errors.Wrap(err, "saving session")
			}
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

	case *serialize.RpcResult:
		obj := message.Obj
		if v, ok := obj.(*serialize.GzipPacked); ok {
			obj = v.Obj
		}

		err := m.writeRPCResponse(int(message.ReqMsgID), obj)
		if err != nil {
			return errors.Wrap(err, "writing RPC response")
		}

	default:
		processed := false
		for _, f := range m.serverRequestHandlers {
			processed = f(message)
			if processed {
				break
			}
		}
		if !processed {
			if m.Warnings != nil {
				m.Warnings <- errors.New("got nonsystem message from server: " + reflect.TypeOf(message).String())
			}
		}
	}

	if (seqNo & 1) != 0 {
		_, err := m.MakeRequest(&serialize.MsgsAck{MsgIds: []int64{int64(msgId)}})
		if err != nil {
			return errors.Wrap(err, "sending ack")
		}
	}

	return nil
}
