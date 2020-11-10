package mtproto

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
	"github.com/xelaj/go-dry"
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/service"
	"github.com/xelaj/mtproto/utils"
)

func (m *MTProto) sendPacket(request tl.Object, response interface{}) (err error) {
	msgID := utils.GenerateMessageId()
	echan := make(chan error)

	requireToAck := false
	if messageRequireToAck(request) {
		m.acks.Put(msgID)
		requireToAck = true
	}

	msg, err := tl.Encode(request)
	if err != nil {
		return err
	}

	data, err := (&service.EncryptedMessage{
		Msg:         msg,
		MsgID:       msgID,
		AuthKeyHash: m.creds.AuthKeyHash,
	}).Serialize(m.sessionID, m.creds.ServerSalt, m.lastSeqNo, m.creds.AuthKey, requireToAck)
	if err != nil {
		return errors.Wrap(err, "serializing message")
	}

	// FIXME:
	// что если:
	// 1. Запрос не требует ответа, но response не nil?
	// 2. Запрос требует ответа, но response nil - нам следует обрабатывать RpcError для него?

	// запрос не требует ответа
	if isNullableResponse(request) {
		pp.Println("nullable:", request)
		go func() {
			echan <- nil
		}()
	} else {
		m.pending.Put(msgID, pendingRequest{
			response: response,
			echan:    echan,
		})
	}

	// этот кусок не часть кодирования так что делаем при отправке
	atomic.AddInt32(&m.lastSeqNo, 2)

	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(len(data)))
	_, err = m.conn.Write(size)
	if err != nil {
		return errors.Wrap(err, "sending data")
	}

	_, err = m.conn.Write(data)
	if err != nil {
		return errors.Wrap(err, "sending request")
	}

	return <-echan
}

func (m *MTProto) readFromConn(ctx context.Context) (data []byte, err error) {
	err = m.conn.SetReadDeadline(time.Now().Add(readTimeout)) // возможно поможет???
	dry.PanicIfErr(err)

	reader := dry.NewCancelableReader(ctx, m.conn)
	// https://core.telegram.org/mtproto/mtproto-transports#abridged
	// что делаем:
	// в conn есть определенный буффер, все что телега присылает, мы сохраняем в буффере, и потом через
	// Read читаем. т.к. маленькие пакеты (до 127 байт)  кодируют длину в 1 байт, а побольше в 4, то
	// мы читаем сначала 1 байт, смотрим, это 0xef или нет, если да, то читаем оставшиеся 3 байта и получаем длину
	//firstByte, err := reader.ReadByte()
	//dry.PanicIfErr(err)
	//
	//sizeInBytes, err := utils.GetPacketLengthMTProtoCompatible([]byte{firstByte})
	//if err == utils.ErrPacketSizeIsBigger {
	//	restOfSize := make([]byte, 3)
	//	n, err := reader.Read(restOfSize)
	//	dry.PanicIfErr(err)
	//	dry.PanicIf(n != 3, fmt.Sprintf("expected read 3 bytes, got %d", n))
	//
	//	sizeInBytes, _ = utils.GetPacketLengthMTProtoCompatible(append([]byte{firstByte}, restOfSize...))
	//
	//	pp.Println(firstByte, restOfSize, sizeInBytes)
	//}

	// https://core.telegram.org/mtproto/mtproto-transports#intermediate
	sizeInBytes := make([]byte, 4)
	n, err := reader.Read(sizeInBytes)
	if err != nil {
		pp.Println(sizeInBytes, err)
		return nil, errors.Wrap(err, "reading length")
	}
	if n != 4 {
		return nil, fmt.Errorf("size is not length of int32, expected 4 bytes, got %d", n)
	}

	size := binary.LittleEndian.Uint32(sizeInBytes)
	// читаем сами данные
	data = make([]byte, int(size))
	n, err = reader.Read(data)
	dry.PanicIfErr(err)
	dry.PanicIf(n != int(size), fmt.Sprintf("expected read %d bytes, got %d", size, n))

	return data, nil
}
