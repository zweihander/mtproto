package mtproto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/service"
)

const (
	// если длина пакета больше или равн 127 слов, то кодируем 4 байтами, 1 это магическое число, оставшиеся 3 — дилна
	magicValueSizeMoreThanSingleByte = 0x7f
)

func isNullableResponse(t tl.Object) bool {
	switch t.(type) {
	case /**service.Ping,*/ *service.Pong, *service.MsgsAck:
		return true
	default:
		return false
	}
}

const (
	readTimeout = 300 * time.Second
)

func CatchResponseErrorCode(data []byte) error {
	if len(data) == 4 {
		code := int(binary.LittleEndian.Uint32(data))
		return &ErrResponseCode{Code: code}
	}
	return nil
}

func IsPacketEncrypted(data []byte) (bool, error) {
	cr := tl.NewReadCursor(bytes.NewBuffer(data))
	authKeyHash, err := cr.PopRawBytes(tl.DoubleLen)
	if err != nil {
		return false, err
	}

	return binary.LittleEndian.Uint64(authKeyHash) != 0, nil
}

func (m *MTProto) decodeRecievedData(data []byte) (*service.EncryptedMessage, error) {
	// проверим, что это не код ошибки
	err := CatchResponseErrorCode(data)
	if err != nil {
		return nil, errors.Wrap(err, "Server response error")
	}

	msg, err := service.DeserializeEncryptedMessage(data, m.creds.AuthKey)

	msgID := msg.MsgID
	atomic.StoreInt64(&m.msgId, msgID)
	atomic.StoreInt32(&m.seqNo, msg.SeqNo)
	mod := msgID & 3
	if mod != 1 && mod != 3 {
		return nil, fmt.Errorf("Wrong bits of message_id: %d", mod)
	}

	return msg, nil
}
