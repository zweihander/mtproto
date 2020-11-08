package tl

import (
	"bytes"
	"fmt"
	"reflect"
)

func Encode(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := encodeValue(NewWriteCursor(buf), reflect.ValueOf(v)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeValue(cur *WriteCursor, value reflect.Value) (err error) {
	if m, ok := value.Interface().(Marshaler); ok {
		return m.MarshalTL(cur)
	}

	switch value.Kind() {
	case reflect.Int32: // reflect.Int, reflect.Uint16, reflect.Uint32:
		err = cur.PutUint(uint32(value.Int()))
	case reflect.Int64:
		err = cur.PutLong(value.Int())
	case reflect.Float64:
		err = cur.PutDouble(value.Float())
	case reflect.Bool:
		err = cur.PutBool(value.Bool())
	case reflect.String:
		err = cur.PutString(value.String())
	case reflect.Ptr, reflect.Interface:
		err = encodeStruct(cur, value.Interface())
	case reflect.Slice:
		if bs, ok := value.Interface().([]byte); ok {
			err = cur.PutMessage(bs)
			break
		}

		err = encodeVector(cur, sliceToInterfaceSlice(value.Interface()))
	default:
		err = fmt.Errorf("unsupported type: %s", value.Type().String())
	}

	return
}

// v must be pointer to struct
func encodeStruct(cur *WriteCursor, v interface{}) error {
	// if reflect.ValueOf(v).IsZero() {
	// 	return fmt.Errorf("zero struct")
	// }

	if m, ok := v.(Marshaler); ok {
		return m.MarshalTL(cur)
	}

	if o, ok := v.(Object); ok {
		cur.PutCRC(o.CRC())
	}

	flag, ok, err := createBitflag(v)
	if err != nil {
		return err
	}

	if ok {
		cur.PutUint(flag)
	}

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("not a pointer")
	}

	val = reflect.Indirect(val)
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("not receiving on struct: %s -> %s", val.Type(), val.Kind())
	}

	vtyp := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if tag, found := vtyp.Field(i).Tag.Lookup("tl"); found {
			info, err := parseTag(tag)
			if err != nil {
				return fmt.Errorf("parsing tag: %w", err)
			}

			if info.encodedInBitflag && vtyp.Field(i).Type.Kind() != reflect.Bool {
				return fmt.Errorf("field '%s': only bool values can be encoded in bitflag", vtyp.Field(i).Name)
			}

			if info.ignore || info.encodedInBitflag {
				continue
			}

			if !val.Field(i).IsZero() {
				if err := encodeValue(cur, val.Field(i)); err != nil {
					return fmt.Errorf("field '%s': %w", vtyp.Field(i).Name, err)
				}
			}

			continue
		}

		// проверка на zero-value, падает на InitConnectionParams.LangPack т.к. он никогда не указывается
		// if val.Field(i).IsZero() {
		// 	return fmt.Errorf("field '%s' have zero value", vtyp.Field(i).Name)
		// }

		if err := encodeValue(cur, val.Field(i)); err != nil {
			return fmt.Errorf("field '%s': %w", vtyp.Field(i).Name, err)
		}
	}

	return nil
}

func createBitflag(v interface{}) (uint32, bool, error) {
	var flag uint32
	haveFlag := false

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		panic("not a pointer")
	}

	val = reflect.Indirect(val)
	if val.Kind() != reflect.Struct {
		return 0, false, fmt.Errorf("not receiving on struct: %s -> %s", val.Type(), val.Kind())
	}

	vtyp := val.Type()
	for i := 0; i < val.NumField(); i++ {
		tag, found := vtyp.Field(i).Tag.Lookup("tl")
		if found {
			info, err := parseTag(tag)
			if err != nil {
				return 0, false, fmt.Errorf("parsing tag: %w", err)
			}

			if info.ignore {
				continue
			}

			haveFlag = true
			if !val.Field(i).IsZero() {
				flag |= 1 << info.index
			}

		}
	}

	return flag, haveFlag, nil
}

func encodeVector(c *WriteCursor, slice []interface{}) (err error) {
	c.PutCRC(CrcVector)
	c.PutUint(uint32(len(slice)))

	for _, item := range slice {
		if err := encodeValue(c, reflect.ValueOf(item)); err != nil {
			return err
		}
		continue

		// switch val := item.(type) {
		// case int8:
		// 	err = c.PutUint(uint32(val))
		// case int16:
		// 	err = c.PutUint(uint32(val))
		// case int32:
		// 	err = c.PutUint(uint32(val))
		// case int64:
		// 	err = c.PutLong(val)
		// case uint8:
		// 	err = c.PutUint(uint32(val))
		// case uint16:
		// 	err = c.PutUint(uint32(val))
		// case uint32:
		// 	err = c.PutUint(val)
		// case uint64:
		// 	err = c.PutLong(int64(val))
		// case bool:
		// 	err = c.PutBool(val)
		// case string:
		// 	err = c.PutString(val)
		// case []byte:
		// 	err = c.PutMessage(val)
		// default:
		// 	err = fmt.Errorf("unserializable type: %T", val)
		// }

		// if err != nil {
		// 	return err
		// }
	}

	return
}
