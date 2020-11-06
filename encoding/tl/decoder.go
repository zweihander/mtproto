package tl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

var (
	objectByCrc map[uint32]Object
	crcByObject map[Object]uint32
	enumCrcs    map[uint32]struct{}
)

func init() {
	objectByCrc = make(map[uint32]Object)
	crcByObject = make(map[Object]uint32)
	enumCrcs = make(map[uint32]struct{})
}

func registerObject(o Object) {
	if _, found := objectByCrc[o.CRC()]; found {
		panic(fmt.Errorf("object with that crc already registered: %d", o.CRC()))
	}

	if another, found := crcByObject[o]; found {
		panic(fmt.Errorf("crc already associated with another object: %T", another))
	}

	objectByCrc[o.CRC()] = o
	crcByObject[o] = o.CRC()
}

func registerEnum(o Object) {
	registerObject(o)
	if _, found := enumCrcs[o.CRC()]; found {
		panic(fmt.Errorf("enum with that crc already registered"))
	}

	enumCrcs[o.CRC()] = struct{}{}
}

func RegisterObjects(obs ...Object) {
	for _, o := range obs {
		registerObject(o)
	}
}

func RegisterEnums(enums ...Object) {
	for _, e := range enums {
		registerEnum(e)
	}
}

func Decode(data []byte, v interface{}) error {
	c := NewReadCursor(bytes.NewBuffer(data))
	if m, ok := v.(Unmarshaler); ok {
		return m.UnmarshalTL(c)
	}

	if o, ok := v.(Object); ok {
		return decodeObject(c, o, false)
	}

	// if obs, ok := v.([]Object); ok {

	// }

	return fmt.Errorf("unsupported type: %T", v)
}

func DecodeRegistered(data []byte) (Object, error) {
	return decodeRegisteredObject(
		NewReadCursor(bytes.NewBuffer(data)),
	)
}

func decodeObject(cur *ReadCursor, o Object, ignoreCRC bool) error {
	if !ignoreCRC {
		crcCode, err := cur.PopCRC()
		if err != nil {
			return fmt.Errorf("read crc: %w", err)
		}

		if crcCode != o.CRC() {
			return fmt.Errorf("invalid crc code: %#v, want: %#v", crcCode, o.CRC())
		}
	}

	value := reflect.ValueOf(o)
	if value.Kind() != reflect.Ptr {
		panic("not a pointer")
	}

	value = reflect.Indirect(value)
	if value.Kind() != reflect.Struct {
		panic("not receiving on struct: " + value.Type().String() + " -> " + value.Kind().String())
	}

	vtyp := value.Type()

	var optionalBitSet uint32
	if haveFlag(o) {
		bitset, err := cur.PopUint()
		if err != nil {
			return fmt.Errorf("read bitset: %w", err)
		}

		optionalBitSet = bitset
	}

	for i := 0; i < value.NumField(); i++ {
		ftyp := value.Field(i).Type()

		if tag, found := vtyp.Field(i).Tag.Lookup(tagName); found {
			info, err := parseFlagTag(tag)
			if err != nil {
				return fmt.Errorf("parse flag: %w", err)
			}

			if optionalBitSet&(1<<info.index) == 0 {
				continue
			}

			if info.encodedInBitflag {
				value.Field(i).Set(reflect.ValueOf(true).Convert(ftyp))
				continue
			}
		}

		switch value.Field(i).Kind() {
		case reflect.Float64:
			val, err := cur.PopDouble()
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(val).Convert(ftyp))
		case reflect.Int64:
			val, err := cur.PopLong()
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(val).Convert(ftyp))
		case reflect.Uint32: // это применимо так же к енумам
			val, err := cur.PopUint()
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(val).Convert(ftyp))
		case reflect.Int32:
			val, err := cur.PopUint()
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(int(val)).Convert(ftyp))
		case reflect.Bool:
			val, err := cur.PopBool()
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(val).Convert(ftyp))
		case reflect.String:
			msg, err := decodeMessage(cur)
			if err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(string(msg)).Convert(ftyp))
		case reflect.Struct:
			fieldValue := reflect.New(ftyp).Elem().Interface().(Object)
			if err := decodeObject(cur, fieldValue, false); err != nil {
				return err
			}

			value.Field(i).Set(reflect.ValueOf(fieldValue).Convert(ftyp))
		case reflect.Slice:
			if _, ok := value.Field(i).Interface().([]byte); ok {
				msg, err := decodeMessage(cur)
				if err != nil {
					return err
				}

				value.Field(i).Set(reflect.ValueOf(msg))
			} else {
				vec, err := decodeVector(cur, ftyp.Elem())
				if err != nil {
					return err
				}

				value.Field(i).Set(reflect.ValueOf(vec).Convert(ftyp))
			}
		case reflect.Ptr:
			if m, ok := value.Field(i).Interface().(Unmarshaler); ok {
				if err := m.UnmarshalTL(cur); err != nil {
					return err
				}
			}

			if o, ok := value.Field(i).Interface().(Object); ok {
				value.Field(i).Set(reflect.New(value.Field(i).Type().Elem()))
				if err := decodeObject(cur, o, false); err != nil {
					return err
				}
			}

			return fmt.Errorf("неизвестная штука: %s", value.Field(i).Type().String())
		case reflect.Interface:
			if !value.Field(i).Type().Implements(reflect.TypeOf((*Object)(nil)).Elem()) {
				panic("can't parse any type, if it don't implement Object")
			}
			field, err := decodeRegisteredObject(cur)
			if err != nil {
				return err
			}

			if !reflect.TypeOf(field).Implements(value.Field(i).Type()) {
				panic("received value " + reflect.TypeOf(field).String() + "; expected " + value.Field(i).Type().String())
			}
			value.Field(i).Set(reflect.ValueOf(field))

		default:
			panic("неизвестная штука: " + value.Field(i).Type().String())
		}
	}

	return nil
}

func decodeRegisteredObject(cur *ReadCursor) (Object, error) {
	crc, err := cur.PopCRC()
	if err != nil {
		return nil, fmt.Errorf("read crc: %w", err)
	}

	o, ok := objectByCrc[crc]
	if !ok {
		return nil, fmt.Errorf("object with crc %#v not found", crc)
	}

	if o == nil {
		panic("nil object")
	}

	if _, isEnum := enumCrcs[crc]; !isEnum {
		err := decodeObject(cur, o, true)
		if err != nil {
			return nil, err
		}
	}

	return o, nil
}

func decodeMessage(c *ReadCursor) ([]byte, error) {
	var firstByte byte
	val := []byte{0}

	c.read(val)
	firstByte = val[0]

	realSize := 0
	lenNumberSize := 0 // сколько байт занимаем число обозначающее длину массива
	if firstByte != FuckingMagicNumber {
		realSize = int(firstByte) // это tinyMessage по сути, первый байт является 8битным числом, которое представляет длину сообщения
		lenNumberSize = 1
	} else {
		// иначе это largeMessage с блядским магитческим числом 0xfe
		realSizeBuf := make([]byte, WordLen-1) // WordLen-1 т.к. 1 байт уже прочитали
		c.read(realSizeBuf)
		realSizeBuf = append(realSizeBuf, 0x0) // добиваем до WordLen

		realSize = int(binary.LittleEndian.Uint32(realSizeBuf))
		lenNumberSize = WordLen
	}

	buf := make([]byte, realSize)
	c.read(buf)
	readLen := lenNumberSize + realSize // lenNumberSize это сколько байт ушло на описание длины а realsize это сколько мы по факту прочитали
	if readLen%WordLen != 0 {
		voidBytes := make([]byte, 4-readLen%WordLen)
		c.read(voidBytes) // читаем оставшиеся пустые байты. пустые, потому что длина слова 4 байта, может остаться 1,2 или 3 лишних байта
		for _, b := range voidBytes {
			if b != 0 {
				// pp.Println(string(buf))
				return nil, fmt.Errorf("some of bytes doesn't equal zero: %#v", voidBytes)
			}
		}
	}

	return buf, nil
}

func decodeVector(c *ReadCursor, as reflect.Type) (interface{}, error) {
	crc, err := c.PopCRC()
	if err != nil {
		return nil, fmt.Errorf("read crc: %w", err)
	}

	if crc != CrcVector {
		return nil, fmt.Errorf("not a vector: %#v, want: %#v", crc, CrcVector)
	}

	size, err := c.PopUint()
	if err != nil {
		return nil, fmt.Errorf("read vector size: %w", err)
	}

	x := reflect.MakeSlice(reflect.SliceOf(as), int(size), int(size))

	for i := 0; i < int(size); i++ {
		var v interface{}

		switch as.Kind() {
		case reflect.Bool:
			val, err := c.PopBool()
			if err != nil {
				return nil, err
			}

			v = val
		case reflect.String:
			msg, err := decodeMessage(c)
			if err != nil {
				return nil, err
			}

			v = string(msg)
		case reflect.Int8, reflect.Int16, reflect.Int32:
			val, err := c.PopUint()
			if err != nil {
				return nil, err
			}

			v = int(val)
		case reflect.Uint8, reflect.Uint16, reflect.Uint32:
			val, err := c.PopUint()
			if err != nil {
				return nil, err
			}

			v = val
		case reflect.Struct:
			var err error
			v, err = decodeRegisteredObject(c)
			if err != nil {
				return nil, err
			}
		case reflect.Int64:
			val, err := c.PopLong()
			if err != nil {
				return nil, err
			}

			v = val
		case reflect.Slice:
			if as.Elem().Kind() == reflect.Uint8 { // []byte
				msg, err := decodeMessage(c)
				if err != nil {
					return nil, err
				}

				v = msg
			} else {
				decodedVec, err := decodeVector(c, as.Elem())
				if err != nil {
					return nil, err
				}

				v = decodedVec
			}
		case reflect.Ptr:
			n := reflect.New(as.Elem()).Interface().(Object)
			if err := decodeObject(c, n, false); err != nil {
				return nil, err
			}

			v = n
		case reflect.Interface:
			if !as.Implements(reflect.TypeOf((*Object)(nil)).Elem()) {
				panic("can't parse any type, if it don't implement TL")
			}

			item, err := decodeRegisteredObject(c)
			if err != nil {
				return nil, err
			}

			if !reflect.TypeOf(item).Implements(as) {
				panic("received value " + reflect.TypeOf(item).String() + "; expected " + as.Elem().String())
			}

			v = item
		default:
			panic("как обрабатывать? " + as.String())
		}

		x.Index(i).Set(reflect.ValueOf(v))
	}

	return x.Interface(), nil
}
