package tl_test

import (
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/serialize"
)

type AccountInstallThemeParams struct {
	Dark   bool       `tl:"flag:0,encoded_in_bitflag"`
	Format string     `tl:"flag:1"`
	Theme  InputTheme `tl:"flag:1"`
}

func (e *AccountInstallThemeParams) CRC() uint32 { return 0x7ae43737 }

type InputTheme interface {
	tl.Object
	ImplementsInputTheme()
}

type InputThemeObj struct {
	Id         int64
	AccessHash int64
}

func (*InputThemeObj) CRC() uint32 { return 0x3c5693e9 }

func (*InputThemeObj) ImplementsInputTheme() {}

type AccountUnregisterDeviceParams struct {
	TokenType int32
	Token     string
	OtherUids []int32
}

func (e *AccountUnregisterDeviceParams) CRC() uint32 { return 0x3076c4bf }

type ResPQ struct {
	Nonce        *serialize.Int128
	ServerNonce  *serialize.Int128
	Pq           []byte
	Fingerprints []int64
}

// func (*ResPQ) CRC() uint32 {
// 	return 0x05162463
// }

// func (r *ResPQ) MarshalTL(w *tl.WriteCursor) error {
// 	if err := r.Nonce.MarshalTL(w); err != nil{
// 		return err
// 	}
// 	if err := r.ServerNonce.MarshalTL(w); err != nil{
// 		return err
// 	}
// 	if err := w.PutMessage(r.Pq); err != nil{
// 		return err
// 	}
// 	if err := w.PutVector(r.Fingerprints); err != nil{
// 		return err
// 	}
// 	return nil
// }
