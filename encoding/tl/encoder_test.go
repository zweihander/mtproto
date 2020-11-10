package tl_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/serialize"
	"github.com/xelaj/mtproto/telegram"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		obj     interface{}
		want    []byte
		wantErr string
	}{
		{
			name: "AccountInstallThemeParams",
			obj: &telegram.AccountInstallThemeParams{
				Dark:   true,
				Format: "abc",
				Theme: &telegram.InputThemeObj{
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
			obj: &telegram.AccountUnregisterDeviceParams{
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
		{
			name: "respq",
			obj: &serialize.ResPQ{
				Nonce: &serialize.Int128{
					big.NewInt(123),
				},
				ServerNonce: &serialize.Int128{
					big.NewInt(321),
				},
				Pq:           []byte{1, 2, 3},
				Fingerprints: []int64{322, 1337},
			},
			want: []byte{
				0x63, 0x24, 0x16, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x7b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x01, 0x41, 0x03, 0x01, 0x02, 0x03, 0x15, 0xc4, 0xb5, 0x1c, 0x02, 0x00, 0x00, 0x00,
				0x42, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x39, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "InitConnectionParams",
			obj: &telegram.InvokeWithLayerParams{
				Layer: int32(322),
				Query: &telegram.InitConnectionParams{
					ApiID:          int32(1337),
					DeviceModel:    "abc",
					SystemVersion:  "def",
					AppVersion:     "123",
					SystemLangCode: "en",
					LangCode:       "en",
					Query:          &telegram.HelpGetConfigParams{},
				},
			},
			want: []byte{
				0x0d, 0x0d, 0x9b, 0xda, 0x42, 0x01, 0x00, 0x00, 0xa9, 0x5e, 0xcd, 0xc1, 0x00, 0x00, 0x00, 0x00,
				0x39, 0x05, 0x00, 0x00, 0x03, 0x61, 0x62, 0x63, 0x03, 0x64, 0x65, 0x66, 0x03, 0x31, 0x32, 0x33,
				0x02, 0x65, 0x6e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x65, 0x6e, 0x00, 0x6b, 0x18, 0xf9, 0xc4,
			},
		},
		{
			name: "access-point-rule",
			obj: &telegram.AccessPointRule{
				PhonePrefixRules: "abc",
				DcId:             1,
				Ips: []telegram.IpPort{
					&telegram.IpPortObj{
						Ipv4: 123,
						Port: 12,
					},
					&telegram.IpPortSecret{
						Ipv4: 321,
						Port: 22,
						Secret: []byte{
							1, 2, 3, 4, 5,
						},
					},
				},
			},
			want: []byte{
				0x5f, 0xb6, 0x79, 0x46, 0x03, 0x61, 0x62, 0x63, 0x01, 0x00, 0x00, 0x00, 0x15, 0xc4, 0xb5, 0x1c,
				0x02, 0x00, 0x00, 0x00, 0x73, 0xad, 0x33, 0xd4, 0x7b, 0x00, 0x00, 0x00, 0x0c, 0x00, 0x00, 0x00,
				0x46, 0x26, 0x98, 0x37, 0x41, 0x01, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x05, 0x01, 0x02, 0x03,
				0x04, 0x05, 0x00, 0x00,
			},
		},
		{
			name: "nil-struct",
			obj: &telegram.AccountPasswordSettings{
				Email: "foo",
				SecureSettings: &telegram.SecureSecretSettings{
					SecureAlgo:     nil,
					SecureSecret:   []byte{1},
					SecureSecretId: 1,
				},
			},
			wantErr: "field 'SecureSettings': field 'SecureAlgo': invalid value",
		},
		{
			name: "nil-interface",
			obj: &telegram.ChannelAdminLogEvent{
				Id:     123,
				Date:   123,
				UserId: 123,
				Action: nil,
			},
			wantErr: "field 'Action': invalid value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tl.Encode(tt.obj)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
					return
				}

				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}
