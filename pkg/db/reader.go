package db

import (
	"errors"
	"io"

	"github.com/amadigan/flit/pkg/schema"
)

const BuiltinKeys = 2
const KeyBuiltinEndOfObject = 0
const KeyBuiltinEmptyKey = 1

var ErrInvalidDocumentId = errors.New("invalid document id")
var ErrDocumentNotFound = errors.New("document not found")

type DBInfo struct {
	Keys  []string
	IdKey string
}

type DBChunk struct {
	Path       string `json:"path"`
	HeaderSize int    `json:"header_size"`
	Size       uint64 `json:"size"`
	FirstDoc   int32  `json:"first_doc"`
	Docs       int32  `json:"docs"`
}

func (c DBInfo) Key(id uint8) string {
	if id < BuiltinKeys {
		return ""
	}
	intId := int(id - BuiltinKeys)
	if intId < 0 || intId >= len(c.Keys) {
		return ""
	}
	return c.Keys[intId]
}

type ValueHeader struct {
	predefined uint8
	nameOff    uint32
	nameLen    uint32
	typ        Type
	off        uint32
	len        uint32
	hdrLen     uint32
	headers    []ValueHeader
	literal    bool
	value      any
	sign       int
}

func (vh ValueHeader) Type() Type {
	return vh.typ
}

func (vh ValueHeader) Length() uint32 {
	return vh.len
}

func (vh ValueHeader) Literal() (any, bool) {
	return vh.value, vh.literal
}

func (vh ValueHeader) Predefined() uint8 {
	return vh.predefined
}

type DocumentCursor interface {
	io.Closer
	Next(schema.DocumentId) ([]ValueHeader, error)
	Open() (ValueCursor, error)
}

type ValueCursor interface {
	Name(int) (string, error)
	Scalar(int) (ValueParser, error)
	ObjectHeader(int) ([]ValueHeader, error)
	Object(int) (ValueCursor, error)
}

type EmptyFieldCursor struct{}

func (EmptyFieldCursor) Name(int) (string, error) {
	return "", ErrMalformedData
}

func (EmptyFieldCursor) Scalar(int) (ValueParser, error) {
	return nil, ErrMalformedData
}

func (EmptyFieldCursor) ObjectHeader(int) ([]ValueHeader, error) {
	return nil, ErrMalformedData
}

func (EmptyFieldCursor) Object(int) (ValueCursor, error) {
	return nil, ErrMalformedData
}
