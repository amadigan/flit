package db

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
)

type InvalidConversionError struct {
	From Type
	To   reflect.Type
}

func (e InvalidConversionError) Error() string {
	return "invalid conversion from " + e.From.String() + " to " + e.To.String()
}

type ValueParser interface {
	Type() Type
	AsBool() (bool, error)
	AsInt8() (int8, error)
	AsInt16() (int16, error)
	AsInt32() (int32, error)
	AsInt64() (int64, error)
	AsUint8() (uint8, error)
	AsUint16() (uint16, error)
	AsUint32() (uint32, error)
	AsUint64() (uint64, error)
	AsFloat32() (float32, error)
	AsFloat64() (float64, error)
	AsString() (string, error)
	AsBytes() ([]byte, error)
	AsReader() (io.Reader, error)
	As(reflect.Type) (reflect.Value, error)
	AsValue() (any, error)
}

func convertScalar(v ValueParser, t reflect.Type) (reflect.Value, error) {
	if t.Kind() == reflect.Ptr {
		val, err := convertScalar(v, t.Elem())
		if err != nil {
			return reflect.Value{}, err
		}

		ptr := reflect.New(t.Elem())
		ptr.Elem().Set(val)
		return ptr, nil
	}

	switch t.Kind() {
	case reflect.Interface:
		if t.NumMethod() == 0 {
			val, err := v.AsValue()
			if err != nil {
				return reflect.Value{}, err
			}

			return reflect.ValueOf(val), nil
		} else {
			return reflect.Value{}, fmt.Errorf("cannot convert to non-empty interface: %s", t.String())
		}
	case reflect.Bool:
		b, err := v.AsBool()
		if err != nil {
			return reflect.Value{}, err
		}
		val := reflect.New(t).Elem()
		val.SetBool(b)
		return val, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := v.AsInt64()
		if err != nil {
			return reflect.Value{}, err
		}
		val := reflect.New(t).Elem()
		val.SetInt(i)
		return val, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := v.AsUint64()
		if err != nil {
			return reflect.Value{}, err
		}
		val := reflect.New(t).Elem()
		val.SetUint(u)
		return val, nil
	case reflect.Float32, reflect.Float64:
		f, err := v.AsFloat64()
		if err != nil {
			return reflect.Value{}, err
		}
		val := reflect.New(t).Elem()
		val.SetFloat(f)
		return val, nil
	case reflect.String:
		s, err := v.AsString()
		if err != nil {
			return reflect.Value{}, err
		}
		val := reflect.New(t).Elem()
		val.SetString(s)
		return val, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			bs, err := v.AsBytes()
			if err != nil {
				return reflect.Value{}, err
			}

			val := reflect.New(t).Elem()
			val.SetBytes(bs)
			return val, nil
		} else {
			return reflect.Value{}, InvalidConversionError{From: v.Type(), To: t}
		}
	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			bs, err := v.AsBytes()
			if err != nil {
				return reflect.Value{}, err
			}

			vout := reflect.New(t).Elem()

			if len(bs) != vout.Len() {
				return reflect.Value{}, fmt.Errorf("cannot convert to array of length %d: value has length %d", vout.Len(), len(bs))
			}

			reflect.Copy(vout, reflect.ValueOf(bs))
			return vout, nil
		} else {
			return reflect.Value{}, InvalidConversionError{From: v.Type(), To: t}
		}
	default:
		return reflect.Value{}, InvalidConversionError{From: v.Type(), To: t}
	}
}

type nullParser struct{}

func (nullParser) Type() Type                               { return TypeNull }
func (nullParser) AsBool() (bool, error)                    { return false, nil }
func (nullParser) AsInt8() (int8, error)                    { return 0, nil }
func (nullParser) AsInt16() (int16, error)                  { return 0, nil }
func (nullParser) AsInt32() (int32, error)                  { return 0, nil }
func (nullParser) AsInt64() (int64, error)                  { return 0, nil }
func (nullParser) AsUint8() (uint8, error)                  { return 0, nil }
func (nullParser) AsUint16() (uint16, error)                { return 0, nil }
func (nullParser) AsUint32() (uint32, error)                { return 0, nil }
func (nullParser) AsUint64() (uint64, error)                { return 0, nil }
func (nullParser) AsFloat32() (float32, error)              { return 0, nil }
func (nullParser) AsFloat64() (float64, error)              { return 0, nil }
func (nullParser) AsString() (string, error)                { return "", nil }
func (nullParser) AsBytes() ([]byte, error)                 { return nil, nil }
func (nullParser) AsReader() (io.Reader, error)             { return bytes.NewReader(nil), nil }
func (nullParser) As(_ reflect.Type) (reflect.Value, error) { return reflect.Value{}, nil }
func (nullParser) AsValue() (any, error)                    { return nil, nil }

type boolParser struct {
	value bool
}

func (p boolParser) Type() Type {
	return TypeBool
}

func (p boolParser) AsBool() (bool, error) {
	return p.value, nil
}

func (p boolParser) AsInt8() (int8, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsInt16() (int16, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsInt32() (int32, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsInt64() (int64, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsUint8() (uint8, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsUint16() (uint16, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsUint32() (uint32, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsUint64() (uint64, error) {
	if p.value {
		return 1, nil
	}
	return 0, nil
}

func (p boolParser) AsFloat32() (float32, error) {
	if p.value {
		return 1.0, nil
	}
	return 0.0, nil
}

func (p boolParser) AsFloat64() (float64, error) {
	if p.value {
		return 1.0, nil
	}
	return 0.0, nil
}

func (p boolParser) AsString() (string, error) {
	if p.value {
		return "true", nil
	}
	return "false", nil
}

func (p boolParser) AsBytes() ([]byte, error) {
	if p.value {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (p boolParser) AsReader() (io.Reader, error) {
	bs, _ := p.AsBytes()
	return bytes.NewReader(bs), nil
}

func (p boolParser) As(t reflect.Type) (reflect.Value, error) {
	return convertScalar(p, t)
}

func (p boolParser) AsValue() (any, error) {
	return p.value, nil
}

type intParser struct {
	value []byte
	sign  int
}

func (p intParser) Type() Type {
	return TypeInt
}

func (p intParser) AsBool() (bool, error) {
	return len(p.value) != 0, nil
}

func (p intParser) AsInt8() (int8, error) {
	if len(p.value) == 0 {
		return 0, nil
	}

	return int8(p.sign * int(p.value[len(p.value)-1])), nil
}

func (p intParser) AsInt16() (int16, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result int16
	for i := 0; i < lp && i < 2; i++ {
		result |= int16(p.value[lp-1-i]) << (8 * i)
	}
	return int16(p.sign) * result, nil
}

func (p intParser) AsInt32() (int32, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result int32
	for i := 0; i < lp && i < 4; i++ {
		result |= int32(p.value[lp-1-i]) << (8 * i)
	}
	return int32(p.sign) * result, nil
}

func (p intParser) AsInt64() (int64, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result int64
	for i := 0; i < lp && i < 8; i++ {
		result |= int64(p.value[lp-1-i]) << (8 * i)
	}
	return int64(p.sign) * result, nil
}

func (p intParser) AsUint8() (uint8, error) {
	if len(p.value) == 0 {
		return 0, nil
	}

	return uint8(p.value[len(p.value)-1]), nil
}

func (p intParser) AsUint16() (uint16, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result uint16
	for i := 0; i < lp && i < 2; i++ {
		result |= uint16(p.value[lp-1-i]) << (8 * i)
	}
	return result, nil
}

func (p intParser) AsUint32() (uint32, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result uint32
	for i := 0; i < lp && i < 4; i++ {
		result |= uint32(p.value[lp-1-i]) << (8 * i)
	}
	return result, nil
}

func (p intParser) AsUint64() (uint64, error) {
	lp := len(p.value)
	if lp == 0 {
		return 0, nil
	}

	var result uint64
	for i := 0; i < lp && i < 8; i++ {
		result |= uint64(p.value[lp-1-i]) << (8 * i)
	}
	return result, nil
}

func (p intParser) AsFloat32() (float32, error) {
	v, err := p.AsInt64()
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}

func (p intParser) AsFloat64() (float64, error) {
	v, err := p.AsInt64()
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

func (p intParser) AsString() (string, error) {
	v, err := p.AsInt64()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(v, 10), nil
}

func (p intParser) AsBytes() ([]byte, error) {
	if len(p.value) == 0 {
		return []byte{0}, nil
	}

	var rv []byte

	switch len(p.value) {
	case 1:
		rv = []byte{p.value[0]}
	case 2:
		rv = []byte{p.value[0], p.value[1]}
	case 3:
		rv = []byte{0, p.value[0], p.value[1], p.value[2]}
	case 4:
		rv = []byte{p.value[0], p.value[1], p.value[2], p.value[3]}
	case 5:
		rv = []byte{0, 0, 0, p.value[0], p.value[1], p.value[2], p.value[3], p.value[4]}
	case 6:
		rv = []byte{0, 0, p.value[0], p.value[1], p.value[2], p.value[3], p.value[4], p.value[5]}
	case 7:
		rv = []byte{0, p.value[0], p.value[1], p.value[2], p.value[3], p.value[4], p.value[5], p.value[6]}
	case 8:
		rv = []byte{p.value[0], p.value[1], p.value[2], p.value[3], p.value[4], p.value[5], p.value[6], p.value[7]}
	default:
		return nil, ErrValueTooLarge
	}

	if p.sign < 0 {
		for i := range rv {
			rv[i] = ^rv[i]
		}
	}

	return rv, nil
}

func (p intParser) AsReader() (io.Reader, error) {
	bs, _ := p.AsBytes()
	return bytes.NewReader(bs), nil
}

func (p intParser) As(t reflect.Type) (reflect.Value, error) {
	return convertScalar(p, t)
}

func (p intParser) AsValue() (any, error) {
	switch len(p.value) {
	case 0:
		return uint8(0), nil
	case 1:
		if p.sign < 0 {
			return p.AsInt8()
		}
		return p.AsUint8()
	case 2:
		if p.sign < 0 {
			return p.AsInt16()
		}
		return p.AsUint16()
	case 3, 4:
		if p.sign < 0 {
			return p.AsInt32()
		}
		return p.AsUint32()
	default:
		if p.sign < 0 {
			return p.AsInt64()
		}
		return p.AsUint64()
	}
}

func (p floatParser) Type() Type {
	return TypeFloat
}

type floatParser struct {
	value []byte
}

func (p floatParser) AsBool() (bool, error) {
	return len(p.value) != 0, nil
}

func (p floatParser) AsInt8() (int8, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return int8(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return int8(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsInt16() (int16, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return int16(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return int16(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsInt32() (int32, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return int32(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return int32(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsInt64() (int64, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return int64(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return int64(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsUint8() (uint8, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return uint8(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return uint8(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsUint16() (uint16, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return uint16(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return uint16(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsUint32() (uint32, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return uint32(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return uint32(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsUint64() (uint64, error) {
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return uint64(f32), err32
	case 8:
		f64, err64 := p.AsFloat64()
		return uint64(f64), err64
	default:
		return 0, ErrValueTooLarge
	}
}

func (p floatParser) AsFloat32() (float32, error) {
	var u32 uint32
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2:
		u32 = uint32(p.value[0])<<8 | uint32(p.value[1])
	case 4:
		u32 = uint32(p.value[0])<<24 | uint32(p.value[1])<<16 | uint32(p.value[2])<<8 | uint32(p.value[3])
	case 8:
		f64, _ := p.AsFloat64()
		return float32(f64), nil
	default:
		return 0, ErrValueTooLarge
	}
	return math.Float32frombits(u32), nil
}

func (p floatParser) AsFloat64() (float64, error) {
	var u64 uint64
	switch len(p.value) {
	case 0:
		return 0, nil
	case 2:
		u64 = uint64(p.value[0])<<8 | uint64(p.value[1])
	case 4:
		u64 = uint64(p.value[0])<<24 | uint64(p.value[1])<<16 | uint64(p.value[2])<<8 | uint64(p.value[3])
	case 8:
		u64 = uint64(p.value[0])<<56 | uint64(p.value[1])<<48 | uint64(p.value[2])<<40 | uint64(p.value[3])<<32 |
			uint64(p.value[4])<<24 | uint64(p.value[5])<<16 | uint64(p.value[6])<<8 | uint64(p.value[7])
	default:
		return 0, ErrValueTooLarge
	}
	return math.Float64frombits(u64), nil
}

func (p floatParser) AsString() (string, error) {
	switch len(p.value) {
	case 0:
		return "0", nil
	case 2, 4:
		f32, _ := p.AsFloat32()
		return strconv.FormatFloat(float64(f32), 'f', -1, 32), nil
	case 8:
		f64, _ := p.AsFloat64()
		return strconv.FormatFloat(f64, 'f', -1, 64), nil
	default:
		return "", ErrValueTooLarge
	}
}

func (p floatParser) AsBytes() ([]byte, error) {
	if len(p.value) <= 4 {
		f32, _ := p.AsFloat32()
		u32 := math.Float32bits(f32)
		return []byte{byte(u32 >> 24), byte(u32 >> 16), byte(u32 >> 8), byte(u32)}, nil
	} else if len(p.value) <= 8 {
		f64, _ := p.AsFloat64()
		u64 := math.Float64bits(f64)
		return []byte{
			byte(u64 >> 56), byte(u64 >> 48), byte(u64 >> 40), byte(u64 >> 32),
			byte(u64 >> 24), byte(u64 >> 16), byte(u64 >> 8), byte(u64),
		}, nil
	}
	return nil, ErrValueTooLarge
}

func (p floatParser) AsReader() (io.Reader, error) {
	bs, _ := p.AsBytes()
	return bytes.NewReader(bs), nil
}

func (p floatParser) As(t reflect.Type) (reflect.Value, error) {
	return convertScalar(p, t)
}

func (p floatParser) AsValue() (any, error) {
	switch len(p.value) {
	case 0:
		return float32(0), nil
	case 2, 4:
		f32, err32 := p.AsFloat32()
		return f32, err32
	case 8:
		f64, err64 := p.AsFloat64()
		return f64, err64
	default:
		return nil, nil
	}
}

type stringParser struct {
	length uint32
	reader io.Reader
}

func (p stringParser) Type() Type {
	return TypeString
}

func (p stringParser) AsBool() (bool, error) {
	s, err := p.AsString()
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(s)
}

func (p stringParser) AsInt8() (int8, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 8)
	if err != nil {
		return 0, err
	}

	return int8(i), nil
}

func (p stringParser) AsInt16() (int16, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		return 0, err
	}

	return int16(i), nil
}

func (p stringParser) AsInt32() (int32, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(i), nil
}

func (p stringParser) AsInt64() (int64, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (p stringParser) AsUint8() (uint8, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	u, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, err
	}

	return uint8(u), nil
}

func (p stringParser) AsUint16() (uint16, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	u, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}

	return uint16(u), nil
}

func (p stringParser) AsUint32() (uint32, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	u, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(u), nil
}

func (p stringParser) AsUint64() (uint64, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}

	return u, nil
}

func (p stringParser) AsFloat32() (float32, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}

	return float32(f), nil
}

func (p stringParser) AsFloat64() (float64, error) {
	s, err := p.AsString()
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

func (p stringParser) AsString() (string, error) {
	if p.length == 0 {
		return "", nil
	}

	bs, err := p.AsBytes()
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func (p stringParser) AsBytes() ([]byte, error) {
	if p.length == 0 {
		return []byte{}, nil
	}

	bs := make([]byte, p.length)
	_, err := io.ReadFull(p.reader, bs)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func (p stringParser) AsReader() (io.Reader, error) {
	if p.length == 0 {
		return bytes.NewReader([]byte{}), nil
	}

	return p.reader, nil
}

var binaryUnmarshalerType = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (p stringParser) As(t reflect.Type) (reflect.Value, error) {
	if t.Kind() == reflect.Ptr {
		v, err := p.As(t.Elem())
		if err != nil {
			return reflect.Value{}, err
		}

		ptr := reflect.New(t.Elem())
		ptr.Elem().Set(v)
		return ptr, nil
	}

	// check if the type supports BinaryUnmarshaler interface
	if t.Implements(binaryUnmarshalerType) {
		bs, err := p.AsBytes()
		if err != nil {
			return reflect.Value{}, err
		}

		val := reflect.New(t)
		err = val.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary(bs)
		if err != nil {
			return reflect.Value{}, err
		}

		return val.Elem(), nil
	}

	// check if the type supports TextUnmarshaler interface
	if t.Implements(textUnmarshalerType) {
		s, err := p.AsString()
		if err != nil {
			return reflect.Value{}, err
		}

		val := reflect.New(t)
		err = val.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
		if err != nil {
			return reflect.Value{}, err
		}

		return val.Elem(), nil
	}

	return convertScalar(p, t)
}

func (p stringParser) AsValue() (any, error) {
	return p.AsString()
}
