package serialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_decompressData(t *testing.T) {
	tests := []struct {
		name    string
		args    []byte
		want    []byte
		wantErr bool
	}{
		{
			name: "test1",
			args: []byte{
				0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x4b, 0x77, 0xe0, 0x36, 0xf6, 0xe0,
				0x63, 0x60, 0x98, 0xb6, 0x71, 0x79, 0x7c, 0xe2, 0xfe, 0xe5, 0xf1, 0xe6, 0xd3, 0x2b, 0xf7, 0x30,
				0x31, 0x30, 0x30, 0x88, 0x1e, 0xd9, 0x2a, 0x23, 0x0c, 0xa4, 0x79, 0x17, 0x6e, 0x97, 0x00, 0x52,
				0x0c, 0x8c, 0x40, 0xcc, 0x67, 0x68, 0x62, 0xa9, 0x67, 0x68, 0x6a, 0xa2, 0x67, 0x68, 0x6e, 0xaa,
				0x67, 0x6a, 0xcc, 0xb0, 0x9b, 0x11, 0x22, 0x2f, 0x40, 0x40, 0x9e, 0x11, 0x2a, 0xaf, 0x6e, 0x64,
				0x60, 0x60, 0x68, 0x65, 0x90, 0x64, 0x64, 0x61, 0x95, 0x66, 0x64, 0x9c, 0x62, 0x95, 0x06, 0xe6,
				0x02, 0x01, 0x2a, 0x91, 0x08, 0xd3, 0x07, 0xb2, 0x97, 0x09, 0xc5, 0x5c, 0x33, 0x73, 0x3d, 0x53,
				0x43, 0x14, 0x7b, 0xf1, 0xc9, 0x33, 0x41, 0xe5, 0xf9, 0x91, 0xe5, 0x0d, 0x4d, 0x0d, 0x91, 0xdd,
				0xc5, 0x84, 0x70, 0x97, 0x99, 0x79, 0xb2, 0x95, 0x81, 0x49, 0xaa, 0x05, 0xc8, 0x5d, 0x46, 0xf8,
				0xdc, 0xc5, 0x4c, 0x9a, 0xbe, 0x24, 0x64, 0xff, 0x30, 0xa3, 0xb8, 0x07, 0x18, 0x4e, 0x86, 0x06,
				0x06, 0xc8, 0xfe, 0xc1, 0x27, 0xcf, 0x08, 0x95, 0xc7, 0x12, 0x8e, 0xc6, 0x84, 0xc2, 0x91, 0x05,
				0x3d, 0x9c, 0x2c, 0x51, 0xc3, 0x11, 0x9f, 0x3c, 0x23, 0x54, 0x1e, 0x8b, 0x7f, 0x4d, 0xf0, 0xd9,
				0xcb, 0x84, 0x69, 0xae, 0xa9, 0x9e, 0xa5, 0x19, 0x03, 0x72, 0x38, 0x92, 0x60, 0x6e, 0x12, 0xb2,
				0x7b, 0x58, 0x31, 0xc2, 0x21, 0x0d, 0xa4, 0xcf, 0x94, 0x50, 0x38, 0x80, 0xf4, 0xf1, 0x5a, 0x1a,
				0x02, 0x83, 0xd5, 0x42, 0xcf, 0xd4, 0x0c, 0x18, 0xc2, 0x46, 0x0c, 0x28, 0xe1, 0x80, 0x53, 0x3e,
				0xb1, 0xa0, 0xcc, 0x58, 0xaf, 0xb8, 0x24, 0x35, 0x47, 0x2f, 0x39, 0x3f, 0x97, 0x81, 0xe1, 0x04,
				0x50, 0x9d, 0x03, 0x2f, 0x33, 0x43, 0x0a, 0x90, 0x0e, 0x30, 0x61, 0x66, 0xe8, 0x00, 0x66, 0x16,
				0x83, 0x52, 0x06, 0x86, 0x07, 0x93, 0x59, 0xc0, 0xf4, 0x1d, 0xa0, 0x41, 0x09, 0xaf, 0x20, 0xe9,
				0x04, 0xa4, 0x96, 0x61, 0x31, 0x13, 0xc3, 0xff, 0xff, 0xff, 0xeb, 0x41, 0x98, 0xe1, 0x95, 0x0a,
				0x58, 0x0c, 0x64, 0x57, 0x83, 0x15, 0x27, 0x98, 0x06, 0x99, 0xa3, 0xe0, 0xc7, 0xc0, 0x30, 0x21,
				0x9e, 0x11, 0xac, 0x5f, 0x40, 0x1d, 0x68, 0x67, 0x46, 0x49, 0x49, 0x41, 0xb1, 0x95, 0xbe, 0x7e,
				0x89, 0x5e, 0x6e, 0xaa, 0x3e, 0x30, 0xee, 0xd3, 0x33, 0xd3, 0xb8, 0xd2, 0xf2, 0x4b, 0x8b, 0x8a,
				0x0b, 0x4b, 0x13, 0x8b, 0x52, 0x19, 0x58, 0x92, 0x32, 0xf3, 0xd2, 0x61, 0x91, 0x2b, 0x00, 0xa1,
				0x00, 0x51, 0xcc, 0xec, 0xfb, 0xd0, 0x03, 0x00, 0x00,
			},
			want: []byte{
				0x67, 0x40, 0x0b, 0x33, 0x48, 0x0e, 0x00, 0x00, 0x96, 0xb1, 0xa7, 0x5f, 0x61, 0xbf, 0xa7, 0x5f,
				0x37, 0x97, 0x79, 0xbc, 0x02, 0x00, 0x00, 0x00, 0x15, 0xc4, 0xb5, 0x1c, 0x13, 0x00, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39,
				0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x37, 0x35, 0x2e, 0x35, 0x33, 0x00, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x10, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39,
				0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x37, 0x35, 0x2e, 0x35, 0x33, 0x00, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30,
				0x31, 0x3a, 0x30, 0x62, 0x32, 0x38, 0x3a, 0x66, 0x32, 0x33, 0x64, 0x3a, 0x66, 0x30, 0x30, 0x31,
				0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a,
				0x30, 0x30, 0x30, 0x61, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x00, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x37,
				0x2e, 0x35, 0x31, 0x00, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x10, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x37,
				0x2e, 0x35, 0x31, 0x00, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x02, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x0f, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x37,
				0x2e, 0x31, 0x35, 0x31, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x01, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30, 0x31, 0x3a, 0x30, 0x36, 0x37, 0x63, 0x3a, 0x30,
				0x34, 0x65, 0x38, 0x3a, 0x66, 0x30, 0x30, 0x32, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30,
				0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x61, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x03, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30,
				0x31, 0x3a, 0x30, 0x36, 0x37, 0x63, 0x3a, 0x30, 0x34, 0x65, 0x38, 0x3a, 0x66, 0x30, 0x30, 0x32,
				0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a,
				0x30, 0x30, 0x30, 0x62, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x00, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00, 0x0f, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x37, 0x35,
				0x2e, 0x31, 0x30, 0x30, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x10, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00, 0x0f, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x37, 0x35,
				0x2e, 0x31, 0x30, 0x30, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x01, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30, 0x31, 0x3a, 0x30, 0x62, 0x32, 0x38, 0x3a, 0x66,
				0x32, 0x33, 0x64, 0x3a, 0x66, 0x30, 0x30, 0x33, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30,
				0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x61, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39,
				0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x37, 0x2e, 0x39, 0x31, 0x00, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39,
				0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x37, 0x2e, 0x39, 0x31, 0x00, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x01, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30,
				0x31, 0x3a, 0x30, 0x36, 0x37, 0x63, 0x3a, 0x30, 0x34, 0x65, 0x38, 0x3a, 0x66, 0x30, 0x30, 0x34,
				0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a,
				0x30, 0x30, 0x30, 0x61, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x02, 0x00, 0x00, 0x00,
				0x04, 0x00, 0x00, 0x00, 0x0e, 0x31, 0x34, 0x39, 0x2e, 0x31, 0x35, 0x34, 0x2e, 0x31, 0x36, 0x35,
				0x2e, 0x39, 0x36, 0x00, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x03, 0x00, 0x00, 0x00,
				0x04, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30, 0x31, 0x3a, 0x30, 0x36, 0x37, 0x63, 0x3a, 0x30,
				0x34, 0x65, 0x38, 0x3a, 0x66, 0x30, 0x30, 0x34, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30,
				0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x62, 0xbb, 0x01, 0x00, 0x00,
				0x0d, 0xa1, 0xb7, 0x18, 0x01, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x27, 0x32, 0x30, 0x30,
				0x31, 0x3a, 0x30, 0x62, 0x32, 0x38, 0x3a, 0x66, 0x32, 0x33, 0x66, 0x3a, 0x66, 0x30, 0x30, 0x35,
				0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x30, 0x30, 0x3a,
				0x30, 0x30, 0x30, 0x61, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x00, 0x00, 0x00, 0x00,
				0x05, 0x00, 0x00, 0x00, 0x0d, 0x39, 0x31, 0x2e, 0x31, 0x30, 0x38, 0x2e, 0x35, 0x36, 0x2e, 0x31,
				0x37, 0x32, 0x00, 0x00, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0xa1, 0xb7, 0x18, 0x10, 0x00, 0x00, 0x00,
				0x05, 0x00, 0x00, 0x00, 0x0d, 0x39, 0x31, 0x2e, 0x31, 0x30, 0x38, 0x2e, 0x35, 0x36, 0x2e, 0x31,
				0x37, 0x32, 0x00, 0x00, 0xbb, 0x01, 0x00, 0x00, 0x0d, 0x61, 0x70, 0x76, 0x33, 0x2e, 0x73, 0x74,
				0x65, 0x6c, 0x2e, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0xc8, 0x00, 0x00, 0x00, 0x40, 0x0d, 0x03, 0x00,
				0x64, 0x00, 0x00, 0x00, 0x50, 0x34, 0x03, 0x00, 0x88, 0x13, 0x00, 0x00, 0x30, 0x75, 0x00, 0x00,
				0xe0, 0x93, 0x04, 0x00, 0x30, 0x75, 0x00, 0x00, 0xdc, 0x05, 0x00, 0x00, 0x60, 0xea, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0xc8, 0x00, 0x00, 0x00, 0x00, 0xa3, 0x02, 0x00, 0xff, 0xff, 0xff, 0x7f,
				0xff, 0xff, 0xff, 0x7f, 0x00, 0xea, 0x24, 0x00, 0xc8, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
				0x80, 0x3a, 0x09, 0x00, 0x05, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x20, 0x4e, 0x00, 0x00,
				0x90, 0x5f, 0x01, 0x00, 0x30, 0x75, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x0d, 0x68, 0x74, 0x74,
				0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x74, 0x2e, 0x6d, 0x65, 0x2f, 0x00, 0x00, 0x03, 0x67, 0x69, 0x66,
				0x0a, 0x66, 0x6f, 0x75, 0x72, 0x73, 0x71, 0x75, 0x61, 0x72, 0x65, 0x00, 0x04, 0x62, 0x69, 0x6e,
				0x67, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decompressData(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("decompressData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
