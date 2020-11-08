package serialize

import (
	"bytes"
	"compress/gzip"
	"io"
)

func decompressData(data []byte) ([]byte, error) {
	// TODO: СТАНДАРТНЫЙ СУКА ПАКЕТ gzip пишет "gzip: invalid header". при этом как я разобрался, в
	//       сам гзип попадает кусок, который находится за миллиард бит от реального сообщения
	//       например: сообщение начинается с 0x1f 0x8b 0x08 0x00 ..., но при этом в сам гзип
	//       отдается кусок, который дальше начала сообщения за 500+ байт
	//! вот ЭТОТ кусок работает. так что наверное не будем трогать, дай бог чтоб работал

	decompressed := make([]byte, 0, 4096)

	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	b := make([]byte, 4096)
	for {
		n, err := gz.Read(b)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		decompressed = append(decompressed, b[0:n]...)
		if n <= 0 {
			break
		}
	}

	return decompressed, nil
	//? это то что я пытался сделать
	// data := d.PopMessage()
	// gz, err := gzip.NewReader(bytes.NewBuffer(data))
	// dry.PanicIfErr(err)

	// decompressed, err := ioutil.ReadAll(gz)
	// dry.PanicIfErr(err)

	// return decompressed
}
