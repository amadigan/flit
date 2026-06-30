package db

import (
	"fmt"
	"reflect"
	"strings"
)

type ExtraField struct {
	Name  string
	Value ValueParser
}

type namedHeader struct {
	index  int
	header ValueHeader
}

func Unmarshal(cursor ValueCursor, headers []ValueHeader, v any, extraFields ...ExtraField) error {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("invalid value type: %T", v)
	}

	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}

	if value.Kind() == reflect.Map {
		if err := unmarshalToMap(cursor, headers, value); err != nil {
			return err
		}

		for _, extra := range extraFields {
			val, err := extra.Value.As(value.Type().Elem())
			if err != nil {
				return err
			}

			value.SetMapIndex(reflect.ValueOf(extra.Name), val)
		}

		return nil
	}

	// TODO this is totally wrong, it needs to iterate over the headers in order, not build a map of them
	// headers must always be iterated in order, because the order of the headers is the order of the values in the cursor
	if value.Kind() == reflect.Struct {
		namedHeaders := make(map[string]namedHeader, len(headers))
		for i, header := range headers {
			name, err := cursor.Name(i)
			if err != nil {
				return err
			}
			namedHeaders[strings.ToLower(name)] = namedHeader{index: i, header: header}
		}

		var namedExtraFields map[string]ValueParser
		if len(extraFields) > 0 {
			namedExtraFields = make(map[string]ValueParser, len(extraFields))
			for _, extra := range extraFields {
				namedExtraFields[strings.ToLower(extra.Name)] = extra.Value
			}
		}

		typ := value.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue // unexported field
			}

			tag := field.Tag.Get("bdoc")
			if tag == "-" {
				continue // skip field
			}

			name := field.Name
			if tag != "" {
				name = tag
			}

			header, ok := namedHeaders[strings.ToLower(name)]
			if !ok {
				scalar, ok := namedExtraFields[strings.ToLower(name)]
				if ok {
					fieldValue := value.Field(i)
					val, err := scalar.As(fieldValue.Type())
					if err != nil {
						return err
					}
					fieldValue.Set(val)
				}

				continue
			}

			fieldValue := value.Field(i)
			switch header.header.typ {
			case TypeObject:
				headers, err := cursor.ObjectHeader(header.index)
				if err != nil {
					return err
				}

				objCursor, err := cursor.Object(header.index)
				if err != nil {
					return err
				}

				obj := reflect.New(fieldValue.Type()).Interface()
				if err := Unmarshal(objCursor, headers, obj); err != nil {
					return err
				}

				fieldValue.Set(reflect.ValueOf(obj).Elem())
			case TypeArray:
				headers, err := cursor.ObjectHeader(header.index)
				if err != nil {
					return err
				}

				arrCursor, err := cursor.Object(header.index)
				if err != nil {
					return err
				}

				arr := reflect.New(fieldValue.Type())
				if err := unmarshalToSlice(arrCursor, headers, arr); err != nil {
					return err
				}

				fieldValue.Set(reflect.ValueOf(arr).Elem())
			default:
				parser, err := cursor.Scalar(header.index)
				if err != nil {
					return err
				}

				val, err := parser.As(fieldValue.Type())
				if err != nil {
					return err
				}
				fieldValue.Set(val)
			}
		}

		return nil
	}

	return fmt.Errorf("unsupported value type: %s", value.Kind().String())
}

func unmarshalToMap(cursor ValueCursor, headers []ValueHeader, v reflect.Value) error {
	if v.Kind() != reflect.Map {
		return fmt.Errorf("invalid value type: %s", v.Kind().String())
	}

	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	for i, header := range headers {
		name, err := cursor.Name(i)
		if err != nil {
			return err
		}

		switch header.typ {
		case TypeObject:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return err
			}

			objCursor, err := cursor.Object(i)
			if err != nil {
				return err
			}

			obj := reflect.New(v.Type().Elem()).Interface()
			if err := Unmarshal(objCursor, headers, obj); err != nil {
				return err
			}

			v.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(obj).Elem())
		case TypeArray:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return err
			}

			arrCursor, err := cursor.Object(i)
			if err != nil {
				return err
			}

			arr := reflect.New(v.Type().Elem())
			if err := unmarshalToSlice(arrCursor, headers, arr); err != nil {
				return err
			}

			v.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(arr).Elem())
		default:
			parser, err := cursor.Scalar(i)
			if err != nil {
				return err
			}
			typeOfElem := v.Type().Elem()
			val, err := parser.As(typeOfElem)
			if err != nil {
				return err
			}

			v.SetMapIndex(reflect.ValueOf(name), val)
		}
	}

	return nil
}

func unmarshalToSlice(cursor ValueCursor, headers []ValueHeader, v reflect.Value) error {
	for i, header := range headers {
		switch header.typ {
		case TypeObject:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return err
			}

			objCursor, err := cursor.Object(i)
			if err != nil {
				return err
			}

			obj := reflect.New(v.Type().Elem()).Interface()
			if err := Unmarshal(objCursor, headers, obj); err != nil {
				return err
			}

			v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
		case TypeArray:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return err
			}

			arrCursor, err := cursor.Object(i)
			if err != nil {
				return err
			}

			arr := reflect.New(v.Type().Elem())
			if err := unmarshalToSlice(arrCursor, headers, arr); err != nil {
				return err
			}

			v.Set(reflect.Append(v, reflect.ValueOf(arr).Elem()))
		default:
			parser, err := cursor.Scalar(i)
			if err != nil {
				return err
			}
			typeOfElem := v.Type().Elem()
			val, err := parser.As(typeOfElem)
			if err != nil {
				return err
			}

			v.Set(reflect.Append(v, val))
		}
	}

	return nil
}

func UnmarshalMap(cursor ValueCursor, headers []ValueHeader) (map[string]any, error) {
	result := make(map[string]any, len(headers))
	for i, header := range headers {
		name, err := cursor.Name(i)
		if err != nil {
			return nil, err
		}

		switch header.typ {
		case TypeObject:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return nil, err
			}

			objCursor, err := cursor.Object(i)
			if err != nil {
				return nil, err
			}

			obj, err := UnmarshalMap(objCursor, headers)
			if err != nil {
				return nil, err
			}

			result[name] = obj
		case TypeArray:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return nil, err
			}

			arrCursor, err := cursor.Object(i)
			if err != nil {
				return nil, err
			}

			arr, err := UnmarshalArray(arrCursor, headers)
			if err != nil {
				return nil, err
			}

			result[name] = arr
		default:
			parser, err := cursor.Scalar(i)
			if err != nil {
				return nil, err
			}

			value, err := parser.AsValue()
			if err != nil {
				return nil, err
			}

			result[name] = value
		}
	}

	return result, nil
}

func UnmarshalArray(cursor ValueCursor, headers []ValueHeader) (any, error) {
	if len(headers) == 0 {
		return nil, nil
	}

	allObjects := true

	result := make([]any, len(headers))
	for i, header := range headers {
		switch header.typ {
		case TypeObject:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return nil, err
			}

			objCursor, err := cursor.Object(i)
			if err != nil {
				return nil, err
			}

			obj, err := UnmarshalMap(objCursor, headers)
			if err != nil {
				return nil, err
			}

			result[i] = obj
		case TypeArray:
			headers, err := cursor.ObjectHeader(i)
			if err != nil {
				return nil, err
			}

			arrCursor, err := cursor.Object(i)
			if err != nil {
				return nil, err
			}

			arr, err := UnmarshalArray(arrCursor, headers)
			if err != nil {
				return nil, err
			}

			result[i] = arr
			allObjects = false
		case TypeNull:
			result[i] = nil
		default:
			parser, err := cursor.Scalar(i)
			if err != nil {
				return nil, err
			}

			value, err := parser.AsValue()
			if err != nil {
				return nil, err
			}

			result[i] = value
			allObjects = false
		}
	}

	if allObjects {
		maps := make([]map[string]any, len(result))
		for i, obj := range result {
			maps[i] = obj.(map[string]any)
		}
		return maps, nil
	}

	typ := reflect.TypeOf(result[0])
	mixed := false
	for _, val := range result {
		if reflect.TypeOf(val) != typ {
			mixed = true
			break
		}
	}

	if !mixed {
		slice := reflect.MakeSlice(reflect.SliceOf(typ), len(result), len(result))
		for i, val := range result {
			slice.Index(i).Set(reflect.ValueOf(val))
		}
		return slice.Interface(), nil
	}

	return result, nil
}
