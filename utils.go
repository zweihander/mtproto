package mtproto

import (
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/service"
)

func messageRequireToAck(msg tl.Object) bool {
	switch msg.(type) {
	case /**service.Ping,*/ *service.MsgsAck:
		return false
	default:
		return true
	}
}
