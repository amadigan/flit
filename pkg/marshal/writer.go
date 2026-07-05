package marshal

import (
	"fmt"
	"reflect"
)

type Writer[T any] interface {
	WriteStartObject() error
	WriteEndObject() error
	WriteStartArray() error
	WriteEndArray() error
	WriteNull() error
	WriteBool(bool) error
	WriteInt8(int8) error
	WriteInt16(int16) error
	WriteInt32(int32) error
	WriteInt64(int64) error
	WriteUint8(uint8) error
	WriteUint16(uint16) error
	WriteUint32(uint32) error
	WriteUint64(uint64) error
	WriteFloat32(float32) error
	WriteFloat64(float64) error
	WriteBytes([]byte) error
	WriteBytesCopy([]byte) error
	WriteString(string) error
	WriteValue(value any) (bool, error)
	WriteField(name string, value any, tag T) (bool, error)
}

type TagParser[T any] func(string) (T, error)

func StringParser[T ~string]() TagParser[T] {
	return func(s string) (T, error) {
		var t T = T(s)
		return t, nil
	}
}

func CachingParser[T any](parser TagParser[T]) TagParser[T] {
	cache := make(map[string]T)

	return func(s string) (T, error) {
		if t, ok := cache[s]; ok {
			return t, nil
		}

		t, err := parser(s)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("failed to parse tag %q: %w", s, err)
		}

		cache[s] = t
		return t, nil
	}
}

type Marshaler[T any] struct {
	TagParser TagParser[T]
	Tag       string
}

func (m Marshaler[T]) Marshal(w Writer[T], value any) error {
	write, err := w.WriteValue(value)
	if !write || err != nil {
		return fmt.Errorf("failed to write value %v: %w", value, err)
	}

	v := reflect.ValueOf(value)

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			break
		}

		v = v.Elem()

		if write, err = w.WriteValue(v.Interface()); !write || err != nil {
			return fmt.Errorf("failed to write value %v: %w", v.Interface(), err)
		}
	}

	return m.marshalValue(w, v)
}

func (m Marshaler[T]) marshalValue(w Writer[T], v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		return m.marshalStruct(w, v)
	case reflect.Slice, reflect.Array:
		return m.marshalSlice(w, v)
	case reflect.Map:
		return m.marshalMap(w, v)
	case reflect.Pointer:
		if v.IsNil() {
			return w.WriteNull()
		}
		return m.marshalValue(w, v.Elem())
	}

	switch v.Kind() {
	case reflect.Bool:
		return w.WriteBool(v.Bool())
	case reflect.Int8:
		return w.WriteInt8(int8(v.Int()))
	case reflect.Int16:
		return w.WriteInt16(int16(v.Int()))
	case reflect.Int32:
		return w.WriteInt32(int32(v.Int()))
	case reflect.Int64:
		return w.WriteInt64(v.Int())
	case reflect.Int:
		return w.WriteInt64(v.Int())
	case reflect.Uint8:
		return w.WriteUint8(uint8(v.Uint()))
	case reflect.Uint16:
		return w.WriteUint16(uint16(v.Uint()))
	case reflect.Uint32:
		return w.WriteUint32(uint32(v.Uint()))
	case reflect.Uint64:
		return w.WriteUint64(v.Uint())
	case reflect.Uint:
		return w.WriteUint64(v.Uint())
	case reflect.Float32:
		return w.WriteFloat32(float32(v.Float()))
	case reflect.Float64:
		return w.WriteFloat64(v.Float())
	case reflect.String:
		return w.WriteString(v.String())
	}

	return fmt.Errorf("unsupported type: %s", v.Type().String())
}

func (m Marshaler[T]) marshalField(w Writer[T], name string, tag T, value reflect.Value) error {
	write, err := w.WriteField(name, value.Interface(), tag)
	if !write || err != nil {
		return err
	}

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			break
		}

		value = value.Elem()

		if write, err = w.WriteField(name, value.Interface(), tag); !write || err != nil {
			return err
		}
	}

	return m.marshalValue(w, value)
}

func (m Marshaler[T]) marshalStruct(w Writer[T], v reflect.Value) error {
	startedObject, err := m.marshalStructFields(w, false, v)
	if err != nil {
		return err
	}

	if startedObject {
		if err := w.WriteEndObject(); err != nil {
			return fmt.Errorf("failed to write end object: %w", err)
		}
	}

	return nil
}

func (m Marshaler[T]) marshalStructFields(w Writer[T], startedObject bool, v reflect.Value) (bool, error) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		var parsed T
		var err error

		if m.Tag != "" {
			parsed, err = m.TagParser(field.Tag.Get(m.Tag))
		}

		if err != nil {
			return startedObject, err
		}

		if field.Anonymous {
			anonValue := fieldValue

			for anonValue.Kind() == reflect.Pointer {
				if anonValue.IsNil() {
					break
				}

				anonValue = anonValue.Elem()
			}

			if anonValue.Kind() == reflect.Struct {
				if started, err := m.marshalStructFields(w, startedObject, anonValue); err != nil {
					return startedObject || started, err
				} else if started {
					startedObject = true
				}

				continue
			}
		}

		if !startedObject {
			if err := w.WriteStartObject(); err != nil {
				return false, fmt.Errorf("failed to write start object: %w", err)
			}
			startedObject = true
		}

		if err := m.marshalField(w, field.Name, parsed, fieldValue); err != nil {
			return startedObject, fmt.Errorf("failed to marshal field %q: %w", field.Name, err)
		}
	}

	return startedObject, nil
}

func (m Marshaler[T]) marshalSlice(w Writer[T], v reflect.Value) error {
	if err := w.WriteStartArray(); err != nil {
		return fmt.Errorf("failed to write start array: %w", err)
	}

	for i := 0; i < v.Len(); i++ {
		if err := m.Marshal(w, v.Index(i).Interface()); err != nil {
			return fmt.Errorf("failed to marshal slice element at index %d: %w", i, err)
		}
	}

	if err := w.WriteEndArray(); err != nil {
		return fmt.Errorf("failed to write end array: %w", err)
	}

	return nil
}

func (m Marshaler[T]) marshalMap(w Writer[T], v reflect.Value) error {
	if err := w.WriteStartObject(); err != nil {
		return fmt.Errorf("failed to write start object: %w", err)
	}

	var emptyTag T

	for _, key := range v.MapKeys() {
		value := v.MapIndex(key)

		if write, err := w.WriteField(key.String(), value.Interface(), emptyTag); err != nil {
			return fmt.Errorf("failed to write field %q: %w", key.String(), err)
		} else if !write {
			continue
		}

		if err := m.marshalValue(w, value); err != nil {
			return fmt.Errorf("failed to marshal map value for key %q: %w", key.String(), err)
		}
	}

	if err := w.WriteEndObject(); err != nil {
		return fmt.Errorf("failed to write end map object: %w", err)
	}

	return nil
}
