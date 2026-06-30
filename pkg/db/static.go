package db

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type StaticString string

var _ ValueParser = StaticString("")

func (s StaticString) Type() Type {
	return TypeString
}

func (s StaticString) As(typ reflect.Type) (reflect.Value, error) {
	return convertScalar(s, typ)
}

func (s StaticString) AsBool() (bool, error) {
	return strconv.ParseBool(string(s))
}

func (s StaticString) AsInt8() (int8, error) {
	i, err := strconv.ParseInt(string(s), 10, 8)
	return int8(i), err
}

func (s StaticString) AsInt16() (int16, error) {
	i, err := strconv.ParseInt(string(s), 10, 16)
	return int16(i), err
}

func (s StaticString) AsInt32() (int32, error) {
	i, err := strconv.ParseInt(string(s), 10, 32)
	return int32(i), err
}

func (s StaticString) AsInt64() (int64, error) {
	return strconv.ParseInt(string(s), 10, 64)
}

func (s StaticString) AsUint8() (uint8, error) {
	i, err := strconv.ParseUint(string(s), 10, 8)
	return uint8(i), err
}

func (s StaticString) AsUint16() (uint16, error) {
	i, err := strconv.ParseUint(string(s), 10, 16)
	return uint16(i), err
}

func (s StaticString) AsUint32() (uint32, error) {
	i, err := strconv.ParseUint(string(s), 10, 32)
	return uint32(i), err
}

func (s StaticString) AsUint64() (uint64, error) {
	return strconv.ParseUint(string(s), 10, 64)
}

func (s StaticString) AsFloat32() (float32, error) {
	f, err := strconv.ParseFloat(string(s), 32)
	return float32(f), err
}

func (s StaticString) AsFloat64() (float64, error) {
	return strconv.ParseFloat(string(s), 64)
}

func (s StaticString) AsBytes() ([]byte, error) {
	return []byte(s), nil
}

func (s StaticString) AsString() (string, error) {
	return string(s), nil
}

func (s StaticString) AsReader() (io.Reader, error) {
	return strings.NewReader(string(s)), nil
}

func (s StaticString) AsValue() (any, error) {
	return string(s), nil
}

type StaticInt int64

var _ ValueParser = StaticInt(0)

func (i StaticInt) Type() Type {
	return TypeInt
}

func (i StaticInt) As(typ reflect.Type) (reflect.Value, error) {
	return convertScalar(i, typ)
}

func (i StaticInt) AsBool() (bool, error) {
	return i != 0, nil
}

func (i StaticInt) AsInt8() (int8, error) {
	return int8(i), nil
}

func (i StaticInt) AsInt16() (int16, error) {
	return int16(i), nil
}

func (i StaticInt) AsInt32() (int32, error) {
	return int32(i), nil
}

func (i StaticInt) AsInt64() (int64, error) {
	return int64(i), nil
}

func (i StaticInt) AsUint8() (uint8, error) {
	return uint8(i), nil
}

func (i StaticInt) AsUint16() (uint16, error) {
	return uint16(i), nil
}

func (i StaticInt) AsUint32() (uint32, error) {
	return uint32(i), nil
}

func (i StaticInt) AsUint64() (uint64, error) {
	return uint64(i), nil
}

func (i StaticInt) AsFloat32() (float32, error) {
	return float32(i), nil
}

func (i StaticInt) AsFloat64() (float64, error) {
	return float64(i), nil
}

func (i StaticInt) AsBytes() ([]byte, error) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(i))
	return bs, nil
}

func (i StaticInt) AsString() (string, error) {
	return strconv.FormatInt(int64(i), 10), nil
}

func (i StaticInt) AsReader() (io.Reader, error) {
	bs, _ := i.AsBytes()
	return bytes.NewReader(bs), nil
}

func (i StaticInt) AsValue() (any, error) {
	return int64(i), nil
}
