package tl

import (
	"bytes"
	"fmt"
	"reflect"
)

func Encode(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := encodeValue(NewWriteCursor(buf), v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeValue(cur *WriteCursor, v interface{}) (err error) {
	if v == nil {
		return fmt.Errorf("nil value")
	}

	if m, ok := v.(Marshaler); ok {
		if err := m.MarshalTL(cur); err != nil {
			return err
		}

		return nil
	}

	switch val := v.(type) {
	case int:
		err = cur.PutUint(uint32(val))
	case int8:
		err = cur.PutUint(uint32(val))
	case int16:
		err = cur.PutUint(uint32(val))
	case int32:
		err = cur.PutUint(uint32(val))
	case int64:
		err = cur.PutLong(val)
	case uint8:
		err = cur.PutUint(uint32(val))
	case uint16:
		err = cur.PutUint(uint32(val))
	case uint32:
		err = cur.PutUint(val)
	case uint64:
		err = cur.PutLong(int64(val))
	case bool:
		err = cur.PutBool(val)
	case string:
		err = cur.PutString(val)
	case []byte:
		err = cur.PutMessage(val)
	default:
		if reflect.ValueOf(v).Kind() == reflect.Slice {
			return encodeVector(cur, sliceToInterfaceSlice(v))
		}

		if reflect.ValueOf(v).Kind() == reflect.Ptr {
			if err := encodeStruct(cur, v); err != nil {
				return fmt.Errorf("encode '%T': %w", v, err)
			}

			return
		}

		return fmt.Errorf("unsupported type: %T", v)
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
		if tag, found := vtyp.Field(i).Tag.Lookup(tagName); found {
			info, err := parseFlagTag(tag)
			if err != nil {
				return fmt.Errorf("parsing tag: %w", err)
			}

			if info.ignore || info.encodedInBitflag {
				continue
			}

			if !val.Field(i).IsZero() {
				if err := encodeValue(cur, val.Field(i).Interface()); err != nil {
					return fmt.Errorf("field '%s': %w", vtyp.Field(i).Name, err)
				}
			}

			continue
		}

		if err := encodeValue(cur, val.Field(i).Interface()); err != nil {
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
		tag, found := vtyp.Field(i).Tag.Lookup(tagName)
		if found {
			info, err := parseFlagTag(tag)
			if err != nil {
				return 0, false, fmt.Errorf("parsing tag: %w", err)
			}

			if info.ignore {
				continue
			}

			haveFlag = true
			flag |= 1 << info.index
		}
	}

	return flag, haveFlag, nil
}

func encodeVector(c *WriteCursor, slice []interface{}) (err error) {
	c.PutCRC(CrcVector)
	c.PutUint(uint32(len(slice)))

	for _, item := range slice {
		switch val := item.(type) {
		case int8:
			err = c.PutUint(uint32(val))
		case int16:
			err = c.PutUint(uint32(val))
		case int32:
			err = c.PutUint(uint32(val))
		case int64:
			err = c.PutLong(val)
		case uint8:
			err = c.PutUint(uint32(val))
		case uint16:
			err = c.PutUint(uint32(val))
		case uint32:
			err = c.PutUint(val)
		case uint64:
			err = c.PutLong(int64(val))
		case bool:
			err = c.PutBool(val)
		case string:
			err = c.PutString(val)
		case []byte:
			err = c.PutMessage(val)
		default:
			err = fmt.Errorf("unserializable type: %T", val)
		}
	}

	return
}
