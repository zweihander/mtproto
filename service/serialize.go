package service

import (
	"math/big"

	"github.com/xelaj/mtproto/encoding/tl"
)

const (
	Int128Len = 4 * 4 // int128 16 байт
	Int256Len = 4 * 8 // int256 32 байт
)

type Int128 struct {
	*big.Int
}

func (i *Int128) MarshalTL(w *tl.WriteCursor) error {
	b, err := bigIntToBytes(i.Int, 128)
	if err != nil {
		return err
	}

	return w.PutRawBytes(b)
}

func (i *Int128) UnmarshalTL(r *tl.ReadCursor) error {
	buf, err := r.PopRawBytes(Int128Len)
	if err != nil {
		return err
	}

	i.Int = big.NewInt(0).SetBytes(buf)
	return nil
}

type Int256 struct {
	*big.Int
}

func (i *Int256) MarshalTL(w *tl.WriteCursor) error {
	b, err := bigIntToBytes(i.Int, 256)
	if err != nil {
		return err
	}

	return w.PutRawBytes(b)
}

func (i *Int256) UnmarshalTL(r *tl.ReadCursor) error {
	buf, err := r.PopRawBytes(Int256Len)
	if err != nil {
		return err
	}

	i.Int = big.NewInt(0).SetBytes(buf)
	return nil
}

// ErrorSessionConfigsChanged это пустой объект, который показывает, что конфигурация сессии изменилась, и нужно создавать новую
type ErrorSessionConfigsChanged struct {
}

func (*ErrorSessionConfigsChanged) CRC() uint32 {
	panic("don't use me")
}

func (ErrorSessionConfigsChanged) Error() string {
	return "session configuration was changed"
}
