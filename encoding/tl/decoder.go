package tl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/k0kubun/pp"
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
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		panic("wtf")
	}

	d := reflect.Indirect(reflect.ValueOf(v))
	switch d.Kind() {
	case reflect.Interface:
		o, err := DecodeRegistered(data)
		if err != nil {
			return fmt.Errorf("decode interface: %w", err)
		}

		d.Set(reflect.ValueOf(o))
		return nil
	case reflect.Slice:
		c := NewReadCursor(bytes.NewReader(data))
		for _, v := range v.([]interface{}) {
			var err error
			v, err = decodeVector(c, reflect.TypeOf(v))
			if err != nil {
				return err
			}
		}
		panic("slice not supported yet")
	case reflect.Array:
		panic("array not supported yet")
	default:
		c := NewReadCursor(bytes.NewReader(data))

		return decode(c, v)
	}

}

func decode(c *ReadCursor, v interface{}) error {
	// if reflect.ValueOf(v).IsNil() {
	// 	o, err := decodeRegisteredObject(c)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	panic("keks")
	// 	_ = o
	// 	return nil
	// }

	if m, ok := v.(Unmarshaler); ok {
		return m.UnmarshalTL(c)
	}

	if o, ok := v.(Object); ok {
		err := decodeObject(c, o, false)
		if err != nil {
			return fmt.Errorf("decode %T: %w", v, err)
		}

		return nil
	}

	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		d := reflect.Indirect(reflect.ValueOf(v))
		if d.Type().Kind() != reflect.Interface {
			panic("keks")
		}

		o, err := decodeRegisteredObject(c)
		pp.Println("decoded_fromiface:", o, err)
		panic("kek")
	}

	return fmt.Errorf("unsupported type: %T", v)
}

func DecodeRegistered(data []byte) (Object, error) {
	ob, err := decodeRegisteredObject(
		NewReadCursor(bytes.NewReader(data)),
	)
	if err != nil {
		pp.Println("failed_decode:", data)
		return nil, fmt.Errorf("decode registered object: %w", err)
	}

	return ob, nil
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
	if haveFlag(value.Interface()) {
		bitset, err := cur.PopUint()
		if err != nil {
			return fmt.Errorf("read bitset: %w", err)
		}

		optionalBitSet = bitset
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)

		if tag, found := vtyp.Field(i).Tag.Lookup("tl"); found {

			info, err := parseTag(tag)
			if err != nil {
				return fmt.Errorf("parse tag: %w", err)
			}

			if optionalBitSet&(1<<info.index) == 0 {
				continue
			}

			if info.encodedInBitflag {
				field.Set(reflect.ValueOf(true).Convert(field.Type()))
				continue
			}
		}

		if err := decodeField(cur, field, field.Type()); err != nil {
			return fmt.Errorf("decode field '%s': %w", vtyp.Field(i).Name, err)
		}
	}

	return nil
}

func decodeField(cur *ReadCursor, field reflect.Value, ftyp reflect.Type) error {
	switch field.Kind() {
	case reflect.Float64:
		val, err := cur.PopDouble()
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(val).Convert(ftyp))
	case reflect.Int64:
		val, err := cur.PopLong()
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(val).Convert(ftyp))
	case reflect.Uint32: // это применимо так же к енумам
		val, err := cur.PopUint()
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(val).Convert(ftyp))
	case reflect.Int32:
		val, err := cur.PopUint()
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(int(val)).Convert(ftyp))
	case reflect.Bool:
		val, err := cur.PopBool()
		if err != nil {
			return fmt.Errorf("pop bool: %w", err)
		}

		field.Set(reflect.ValueOf(val).Convert(ftyp))
	case reflect.String:
		msg, err := decodeMessage(cur)
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(string(msg)).Convert(ftyp))
	case reflect.Struct:
		fieldValue := reflect.New(ftyp).Elem().Interface()
		if err := decode(cur, fieldValue); err != nil {
			return err
		}

		field.Set(reflect.ValueOf(fieldValue).Convert(ftyp))
	case reflect.Slice:
		if _, ok := field.Interface().([]byte); ok {
			msg, err := decodeMessage(cur)
			if err != nil {
				return err
			}

			field.Set(reflect.ValueOf(msg))
		} else {
			vec, err := decodeVector(cur, ftyp.Elem())
			if err != nil {
				return err
			}

			field.Set(reflect.ValueOf(vec).Convert(ftyp))
		}
	case reflect.Ptr:
		switch v := field.Interface().(type) {
		case Unmarshaler:
			fieldValue := reflect.New(reflect.Indirect(reflect.Zero(ftyp.Elem())).Type())

			if m, ok := fieldValue.Interface().(Unmarshaler); ok {
				// fmt.Println("unmarshalling!")
				if err := m.UnmarshalTL(cur); err != nil {
					return err
				}
			} else {
				panic("badbad")
			}

			field.Set(fieldValue)
		// case Object:
		// 	panic("unsupported sry")
		// 	value.Field(i).Set(reflect.New(value.Field(i).Type().Elem()))
		// 	if err := decodeObject(cur, o, false); err != nil {
		// 		return err
		// 	}
		default:
			err := fmt.Errorf("неизвестная штука: %T", v)
			panic(err)
		}
	case reflect.Interface:
		// if !value.Field(i).Type().Implements(reflect.TypeOf((*Object)(nil)).Elem()) {
		// 	panic("can't parse any type, if it don't implement Object")
		// }

		if err := decode(cur, field.Interface()); err != nil {
			return err
		}

		// if !reflect.TypeOf(field).Implements(value.Field(i).Type()) {
		// 	panic("received value " + reflect.TypeOf(field).String() + "; expected " + value.Field(i).Type().String())
		// }
		// value.Field(i).Set(reflect.ValueOf(field))

	default:
		panic("неизвестная штука: " + field.Type().String())
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

	if m, ok := o.(Unmarshaler); ok {
		return o, m.UnmarshalTL(cur)
	}

	if _, isEnum := enumCrcs[crc]; !isEnum {
		err := decodeObject(cur, o, true)
		if err != nil {
			return nil, fmt.Errorf("decode %T: %w", o, err)
		}
	}

	return o, nil
}

func decodeMessage(c *ReadCursor) ([]byte, error) {
	var firstByte byte
	val := []byte{0}

	if err := c.read(val); err != nil {
		return nil, err
	}

	firstByte = val[0]

	realSize := 0
	lenNumberSize := 0 // сколько байт занимаем число обозначающее длину массива
	if firstByte != FuckingMagicNumber {
		realSize = int(firstByte) // это tinyMessage по сути, первый байт является 8битным числом, которое представляет длину сообщения
		lenNumberSize = 1
	} else {
		// иначе это largeMessage с блядским магитческим числом 0xfe
		realSizeBuf := make([]byte, WordLen-1) // WordLen-1 т.к. 1 байт уже прочитали
		if err := c.read(realSizeBuf); err != nil {
			return nil, err
		}

		realSizeBuf = append(realSizeBuf, 0x0) // добиваем до WordLen

		realSize = int(binary.LittleEndian.Uint32(realSizeBuf))
		lenNumberSize = WordLen
	}

	buf := make([]byte, realSize)
	if err := c.read(buf); err != nil {
		return nil, err
	}

	readLen := lenNumberSize + realSize // lenNumberSize это сколько байт ушло на описание длины а realsize это сколько мы по факту прочитали
	if readLen%WordLen != 0 {
		voidBytes := make([]byte, 4-readLen%WordLen)
		if err := c.read(voidBytes); err != nil { // читаем оставшиеся пустые байты. пустые, потому что длина слова 4 байта, может остаться 1,2 или 3 лишних байта
			return nil, err
		}

		for _, b := range voidBytes {
			if b != 0 {
				return nil, fmt.Errorf("some of bytes doesn't equal zero: %#v", voidBytes)
			}
		}
	}

	return buf, nil
}

// decode []Object direct
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
			// n := reflect.New(as.Elem()).Interface().(Object)
			// if err := decodeObject(c, n, false); err != nil {
			// 	return nil, err
			// }
			n := reflect.New(as.Elem()).Interface()
			if err := decode(c, n); err != nil {
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
