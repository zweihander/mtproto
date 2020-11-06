package tl_test

import (
	"reflect"
	"testing"

	"github.com/xelaj/mtproto/encoding/tl"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		obj     interface{}
		want    []byte
		wantErr bool
	}{
		{
			name: "AccountInstallThemeParams",
			obj: &AccountInstallThemeParams{
				Dark:   true,
				Format: "abc",
				Theme: &InputThemeObj{
					Id:         123,
					AccessHash: 321,
				},
			},
			want: []byte{
				0x37, 0x37, 0xe4, 0x7a, 0x03, 0x00, 0x00, 0x00, 0x03, 0x61, 0x62, 0x63, 0xe9, 0x93, 0x56, 0x3c,
				0x7b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "AccountUnregisterDeviceParams",
			obj: &AccountUnregisterDeviceParams{
				TokenType: 1,
				Token:     "foo",
				OtherUids: []int32{
					1337, 228, 322,
				},
			},
			want: []byte{
				0xbf, 0xc4, 0x76, 0x30, 0x01, 0x00, 0x00, 0x00, 0x03, 0x66, 0x6f, 0x6f, 0x15, 0xc4, 0xb5, 0x1c,
				0x03, 0x00, 0x00, 0x00, 0x39, 0x05, 0x00, 0x00, 0xe4, 0x00, 0x00, 0x00, 0x42, 0x01, 0x00, 0x00,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tl.Encode(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
