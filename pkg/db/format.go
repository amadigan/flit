package db

import "fmt"

type Type uint8

const (
	TypeNull   Type = 0x01
	TypeBool   Type = 0x02
	TypeInt    Type = 0x03
	TypeFloat  Type = 0x04
	TypeString Type = 0x05
	TypeObject Type = 0x06
	TypeArray  Type = 0x07
)

func (t Type) String() string {
	switch t {
	case TypeNull:
		return "null"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeObject:
		return "object"
	case TypeArray:
		return "array"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

const (
	LitNull        uint8 = 0x01
	LitFalse       uint8 = 0x02
	LitTrue        uint8 = 0x0A
	LitIntZero     uint8 = 0x03
	LitFloatZero   uint8 = 0x04
	LitStringEmpty uint8 = 0x05
	LitObjectEmpty uint8 = 0x06
	LitArrayEmpty  uint8 = 0x07
	EndOfObject    uint8 = 0x80
	EmptyKey       uint8 = 0x81
)

var ErrTruncatedData = fmt.Errorf("truncated data")

var ErrMalformedData = fmt.Errorf("malformed data")

var ErrValueTooLarge = fmt.Errorf("value too large: %w", ErrMalformedData)
