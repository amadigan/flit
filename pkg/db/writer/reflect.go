package writer

import (
	"encoding"
	"reflect"
	"slices"
	"strings"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/marshal"
)

type documentEncoder struct {
	keys      map[string]uint8
	marshaler marshal.Marshaler[string]
	stack     []stackFrame
	fieldName string
	done      bool
}

type stackFrame struct {
	fields []Field
	name   string
}

func MarshalDocument(keys map[string]uint8, value any) ([]Field, error) {
	marshaler := marshal.Marshaler[string]{
		Tag:       "bdoc",
		TagParser: marshal.StringParser[string](),
	}
	encoder := &documentEncoder{
		keys:      keys,
		marshaler: marshaler,
	}

	if err := marshaler.Marshal(encoder, value); err != nil {
		return nil, err
	}

	if len(encoder.stack) != 1 {
		return nil, db.ErrMalformedData
	}

	return encoder.stack[0].fields, nil
}

func (w *documentEncoder) WriteStartObject() error {
	w.stack = append(w.stack, stackFrame{name: w.fieldName})
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteEndObject() error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	if len(w.stack) == 1 {
		if w.done {
			return db.ErrMalformedData
		}
		w.done = true

		return nil
	}

	frame := w.stack[len(w.stack)-1]
	w.stack = w.stack[:len(w.stack)-1]

	field, err := NewObjectField(w.keys, frame.name, frame.fields)
	if err != nil {
		return err
	}

	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)

	return nil
}

func (w *documentEncoder) WriteStartArray() error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}
	w.stack = append(w.stack, stackFrame{name: w.fieldName})
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteEndArray() error {
	if len(w.stack) < 2 {
		return db.ErrMalformedData
	}

	frame := w.stack[len(w.stack)-1]
	w.stack = w.stack[:len(w.stack)-1]

	field, err := NewArrayField(frame.name, frame.fields)
	if err != nil {
		return err
	}

	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)

	return nil
}

func (w *documentEncoder) WriteNull() error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewNullField(w.fieldName)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteBool(value bool) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewBoolField(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteInt8(value int8) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewInt8Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteInt16(value int16) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewInt16Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteInt32(value int32) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewInt32Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteInt64(value int64) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewInt64Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteUint8(value uint8) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewUint8Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteUint16(value uint16) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewUint16Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteUint32(value uint32) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewUint32Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteUint64(value uint64) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewUint64Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteFloat32(value float32) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewFloat32Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteFloat64(value float64) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewFloat64Field(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteString(value string) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewStringField(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteBytes(value []byte) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	field := NewBytesField(w.fieldName, value)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteBytesCopy(value []byte) error {
	if len(w.stack) == 0 {
		return db.ErrMalformedData
	}

	copy := slices.Clone(value)
	field := NewBytesField(w.fieldName, copy)
	w.stack[len(w.stack)-1].fields = append(w.stack[len(w.stack)-1].fields, field)
	w.fieldName = ""

	return nil
}

func (w *documentEncoder) WriteValue(value any) (bool, error) {
	if len(w.stack) == 0 {
		return true, nil
	}

	if binaryMarshaler, ok := value.(encoding.BinaryMarshaler); ok {
		data, err := binaryMarshaler.MarshalBinary()
		if err != nil {
			return false, err
		}

		return false, w.WriteBytes(data)
	}

	if textMarshaler, ok := value.(encoding.TextMarshaler); ok {
		data, err := textMarshaler.MarshalText()
		if err != nil {
			return false, err
		}

		return false, w.WriteString(string(data))
	}

	return true, nil
}

type zeroable interface {
	IsZero() bool
}

func (w *documentEncoder) WriteField(name string, value any, tag string) (bool, error) {
	if len(w.stack) == 0 {
		return false, db.ErrMalformedData
	}

	if tag == "-" {
		return false, nil
	}

	if zeroable, ok := value.(zeroable); ok {
		if zeroable.IsZero() {
			return false, nil
		}
	}

	// check for zero values of primitive types
	if value == nil {
		return false, nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Bool:
		if !v.Bool() {
			return false, nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() == 0 {
			return false, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() == 0 {
			return false, nil
		}
	case reflect.Float32, reflect.Float64:
		if v.Float() == 0 {
			return false, nil
		}
	case reflect.String:
		if v.Len() == 0 {
			return false, nil
		}
	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			return false, nil
		}
	case reflect.Pointer:
		if v.IsNil() {
			return false, nil
		}
	case reflect.Interface:
		if v.IsNil() {
			return false, nil
		}
	}

	w.fieldName = strings.ToLower(name)

	if tag != "" {
		w.fieldName = tag
	}

	return w.WriteValue(value)
}
