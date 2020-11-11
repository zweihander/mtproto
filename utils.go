package mtproto

import (
	"math/rand"
	"time"

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

func generateSessionID() int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63() // nolint: gosec потому что начерта?
}

// GenerateMessageID отдает по сути unix timestamp но ужасно специфическим образом
// TODO: нахуя нужно битовое и на -4??
func generateMessageID() int64 {
	const billion = 1000 * 1000 * 1000
	unixnano := time.Now().UnixNano()
	seconds := unixnano / billion
	nanoseconds := unixnano % billion
	return (seconds << 32) | (nanoseconds & -4)
}