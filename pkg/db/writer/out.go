package writer

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/amadigan/flit/pkg/db"
)

type Field struct {
	typ     uint8
	name    string
	content [][]byte
	len     int
}

func (f Field) writeField(w io.ByteWriter) error {
	if f.len > 0x3FFFFFF {
		return fmt.Errorf("field %q length %d exceeds maximum length of %d", f.name, f.len, 0x3FFFFFF)
	} else if f.len > 0x3FFFF {
		w.WriteByte(f.typ | 0xC0)
		w.WriteByte(uint8(f.len >> 16))
		w.WriteByte(uint8((f.len >> 8) & 0xFF))
		w.WriteByte(uint8(f.len & 0xFF))
	} else if f.len > 0x3FF {
		w.WriteByte(f.typ | 0xA0 | (uint8(f.len>>13) & 0x18))
		w.WriteByte(uint8(f.len >> 8))
		w.WriteByte(uint8(f.len & 0xFF))
	} else if f.len > 0xF {
		w.WriteByte(f.typ | 0x80 | (uint8(f.len>>5) & 0x18))
		w.WriteByte(uint8(f.len & 0xFF))
	} else {
		w.WriteByte(f.typ | (uint8(f.len) << 3))
	}

	return nil
}

const MaxNameLength = 0x3FFF

func BuildObject(keys map[string]uint8, fields []Field) ([][]byte, int, error) {
	// write header
	var hdrWriter bytes.Buffer

	for _, field := range fields {
		if key, predefined := keys[field.name]; predefined {
			hdrWriter.WriteByte(key)
		} else if len(field.name) > MaxNameLength {
			return nil, 0, fmt.Errorf("field name %q exceeds maximum length of %d", field.name, MaxNameLength)
		} else {
			nameLen := len(field.name)
			// log.Printf("writing field %q with name length %d and content length %d", field.name, nameLen, field.len)
			if nameLen > 0x3F {
				hdrWriter.WriteByte(0x40 | uint8(nameLen>>8))
				hdrWriter.WriteByte(uint8(nameLen & 0xFF))
			} else {
				hdrWriter.WriteByte(uint8(nameLen))
			}
		}

		if err := field.writeField(&hdrWriter); err != nil {
			return nil, 0, err
		}
	}

	hdrWriter.WriteByte(db.EndOfObject)

	blocks := [][]byte{hdrWriter.Bytes()}
	totalSize := len(blocks[0])

	for _, field := range fields {
		if _, predefined := keys[field.name]; !predefined && len(field.name) > 0 {
			bs := []byte(field.name)
			blocks = append(blocks, bs)
			totalSize += len(bs)
		}

		blocks = append(blocks, field.content...)
		for _, content := range field.content {
			totalSize += len(content)
		}
	}

	return blocks, totalSize, nil
}

func NewObjectField(predefinedKeys map[string]uint8, name string, fields []Field) (Field, error) {
	f := Field{name: name, typ: uint8(db.TypeObject)}

	if len(fields) == 0 {
		return f, nil
	}

	blocks, totalSize, err := BuildObject(predefinedKeys, fields)
	if err != nil {
		return f, err
	}

	f.content = blocks
	f.len = totalSize

	return f, nil
}

func NewArrayField(name string, fields []Field) (Field, error) {
	f := Field{name: name, typ: uint8(db.TypeArray)}

	var hdrWriter bytes.Buffer

	for _, field := range fields {
		if err := field.writeField(&hdrWriter); err != nil {
			return f, err
		}
	}

	hdrWriter.WriteByte(db.EndOfObject)

	blocks := [][]byte{hdrWriter.Bytes()}
	totalSize := len(blocks[0])

	for _, field := range fields {
		blocks = append(blocks, field.content...)
		for _, content := range field.content {
			totalSize += len(content)
		}
	}

	f.content = blocks
	f.len = totalSize

	return f, nil
}

func NewNullField(name string) Field {
	return Field{name: name, typ: db.LitNull}
}

func NewBoolField(name string, value bool) Field {
	f := Field{name: name}

	if value {
		f.typ = db.LitTrue
	} else {
		f.typ = db.LitFalse
	}

	return f
}

func NewInt8Field(name string, value int8) Field {
	f := Field{name: name, typ: uint8(db.TypeInt)}

	if value > 0 {
		f.content = [][]byte{{byte(value)}}
		f.len = 1
	} else if value < 0 {
		f.content = [][]byte{{byte(-value)}}
		f.len = 1
		f.typ |= 0x40
	}

	return f
}

func NewInt16Field(name string, value int16) Field {
	f := Field{name: name, typ: uint8(db.TypeInt)}

	if value == 0 {
		return f
	}

	if value < 0 {
		f.typ |= 0x40
		value = -value
	}

	if value <= 0xFF {
		f.content = [][]byte{{byte(value)}}
		f.len = 1
	} else {
		f.content = [][]byte{{byte(value >> 8), byte(value & 0xFF)}}
		f.len = 2
	}

	return f
}

func NewInt32Field(name string, value int32) Field {
	f := Field{name: name, typ: uint8(db.TypeInt)}

	if value == 0 {
		return f
	}

	if value < 0 {
		f.typ |= 0x40
		value = -value
	}

	if value <= 0xFF {
		f.content = [][]byte{{byte(value)}}
		f.len = 1
	} else if value <= 0xFFFF {
		f.content = [][]byte{{byte(value >> 8), byte(value & 0xFF)}}
		f.len = 2
	} else if value <= 0xFFFFFF {
		f.content = [][]byte{{byte(value >> 16), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 3
	} else {
		f.content = [][]byte{{byte(value >> 24), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 4
	}

	return f
}

func NewInt64Field(name string, value int64) Field {
	f := Field{name: name, typ: uint8(db.TypeInt)}

	if value == 0 {
		return f
	}

	if value < 0 {
		f.typ |= 0x40
		value = -value
	}

	if value <= 0xFF {
		f.content = [][]byte{{byte(value)}}
		f.len = 1
	} else if value <= 0xFFFF {
		f.content = [][]byte{{byte(value >> 8), byte(value & 0xFF)}}
		f.len = 2
	} else if value <= 0xFFFFFF {
		f.content = [][]byte{{byte(value >> 16), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 3
	} else if value <= 0xFFFFFFFF {
		f.content = [][]byte{{byte(value >> 24), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 4
	} else if value <= 0xFFFFFFFFFFFF {
		f.content = [][]byte{{byte(value >> 40), byte((value >> 32) & 0xFF), byte((value >> 24) & 0xFF), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 5 // 6 byte integer lengths are mapped to a value of 5
	} else {
		f.content = [][]byte{{byte(value >> 56), byte((value >> 48) & 0xFF), byte((value >> 40) & 0xFF), byte((value >> 32) & 0xFF), byte((value >> 24) & 0xFF), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		f.len = 6 // 8 byte integer lengths are mapped to a value of 6
	}

	return f
}

func NewUint8Field(name string, value uint8) Field {
	return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{value}}, len: 1}
}

func NewUint16Field(name string, value uint16) Field {
	if value <= 0xFF {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value)}}, len: 1}
	} else {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value >> 8), byte(value & 0xFF)}}, len: 2}
	}
}

func NewUint32Field(name string, value uint32) Field {
	if value <= 0xFF {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value)}}, len: 1}
	} else if value <= 0xFFFF {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value >> 8), byte(value & 0xFF)}}, len: 2}
	} else if value <= 0xFFFFFF {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value >> 16), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}, len: 3}
	} else {
		return Field{name: name, typ: uint8(db.TypeInt), content: [][]byte{{byte(value >> 24), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}, len: 4}
	}
}

func NewUint64Field(name string, value uint64) Field {
	if value > 0x7FFFFFFFFFFFFFFF {
		f := Field{name: name, typ: uint8(db.TypeInt)}
		f.len = 6
		f.content = [][]byte{{byte(value >> 56), byte((value >> 48) & 0xFF), byte((value >> 40) & 0xFF), byte((value >> 32) & 0xFF), byte((value >> 24) & 0xFF), byte((value >> 16) & 0xFF), byte((value >> 8) & 0xFF), byte(value & 0xFF)}}
		return f
	}
	return NewInt64Field(name, int64(value))
}

func NewFloat32Field(name string, value float32) Field {
	f := Field{name: name, typ: uint8(db.TypeFloat)}
	if value == 0 {
		return f
	}

	bits := math.Float32bits(value)
	f.content = [][]byte{{byte(bits >> 24), byte((bits >> 16) & 0xFF), byte((bits >> 8) & 0xFF), byte(bits & 0xFF)}}
	f.len = 4

	return f
}

func NewFloat64Field(name string, value float64) Field {
	f := Field{name: name, typ: uint8(db.TypeFloat)}
	if value == 0 {
		return f
	}

	bits := math.Float64bits(value)
	f.content = [][]byte{{byte(bits >> 56), byte((bits >> 48) & 0xFF), byte((bits >> 40) & 0xFF), byte((bits >> 32) & 0xFF), byte((bits >> 24) & 0xFF), byte((bits >> 16) & 0xFF), byte((bits >> 8) & 0xFF), byte(bits & 0xFF)}}
	f.len = 6

	return f
}

func NewStringField(name string, value string) Field {
	f := Field{name: name, typ: uint8(db.TypeString)}
	if value == "" {
		return f
	}

	f.content = [][]byte{[]byte(value)}
	f.len = len(value)

	return f
}

func NewBytesField(name string, value []byte) Field {
	f := Field{name: name, typ: uint8(db.TypeString)}
	if len(value) == 0 {
		return f
	}

	f.content = [][]byte{value}
	f.len = len(value)

	return f
}
