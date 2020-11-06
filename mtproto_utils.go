package mtproto

import (
	"fmt"
	"reflect"

	"github.com/xelaj/go-dry"

	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/utils"
)

// мелкие методы, которые сделаны для понимания алгоритмов и кода вцелом

// waitAck добавляет в список id сообщения, которому нужно подтверждение
// возвращает true, если ранее этого id не было
func (m *MTProto) waitAck(msgID int64) bool {
	_, ok := m.idsToAck[msgID]
	m.idsToAck[msgID] = struct{}{}
	return !ok
}

// gotAck удаляет элемент из списка id сообщений, на который ожидается ack.
// возвращается true, если id был найден
func (m *MTProto) gotAck(msgID int64) bool {
	m.idsToAckMutex.Lock()
	_, ok := m.idsToAck[msgID]
	delete(m.idsToAck, msgID)
	m.idsToAckMutex.Unlock()
	return ok
}

// resetAck сбрасывает целиком список сообщений, которым нужен ack
func (m *MTProto) resetAck() {
	m.idsToAck = make(map[int64]struct{})
}

// получает текущий идентификатор сессии
func (m *MTProto) GetSessionID() int64 {
	return m.sessionId
}

// Получает lastSeqNo
func (m *MTProto) GetLastSeqNo() int32 {
	return m.lastSeqNo
}

// получает соль
func (m *MTProto) GetServerSalt() int64 {
	return m.serverSalt
}

// получает ключ авторизации
func (m *MTProto) GetAuthKey() []byte {
	return m.authKey
}

func (m *MTProto) SetAuthKey(key []byte) {
	m.authKey = key
	m.authKeyHash = utils.AuthKeyHash(m.authKey)
}

func (m *MTProto) MakeRequest(msg tl.Object) (tl.Object, error) {
	return m.makeRequest(msg, nil)
}

func (m *MTProto) MakeRequestAsSlice(msg tl.Object, as reflect.Type) (tl.Object, error) {
	return m.makeRequest(msg, as)
}

func (m *MTProto) recoverGoroutine() {
	if r := recover(); r != nil {
		if m.RecoverFunc != nil {
			fmt.Println(dry.StackTrace(0))
			m.RecoverFunc(r)
		} else {
			panic(r)
		}
	}
}

func (m *MTProto) AddCustomServerRequestHandler(handler customHandlerFunc) {
	m.serverRequestHandlers = append(m.serverRequestHandlers, handler)
}
