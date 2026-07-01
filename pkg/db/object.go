package db

import (
	"fmt"
	"io"
	"log"
)

var intLengthMap = []uint32{0, 1, 2, 3, 4, 6, 8}

func parseLength(r io.Reader, typ uint8, buf []byte) (uint32, int, error) {
	if typ&0x80 != 0 {
		vlen := int((typ&0x60)>>5) + 1
		if vlen > 3 {
			return 0, 0, ErrValueTooLarge
		}

		if _, err := io.ReadFull(r, buf[:vlen]); err != nil {
			if err == io.EOF {
				return 0, 0, ErrTruncatedData
			}
			return 0, 0, err
		}

		switch vlen {
		case 3:
			return uint32((uint32(typ)&0x18)<<21 | uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])), vlen, nil
		case 2:
			return uint32((uint32(typ)&0x18)<<13 | uint32(buf[0])<<8 | uint32(buf[1])), vlen, nil
		case 1:
			return uint32((uint32(typ)&0x18)<<5 | uint32(buf[0])), vlen, nil
		default:
			return 0, vlen, ErrMalformedData
		}
	}

	return uint32(typ&0x78) >> 3, 0, nil
}

func parseArrayHeader(r io.Reader) ([]ValueHeader, int, error) {
	var headers []ValueHeader
	buf := make([]byte, 4)
	read := 0
	offset := 0

	for i := 0; ; i++ {
		headers = append(headers, ValueHeader{})

		if n, err := headers[i].parse(r, buf); err != nil {
			return headers, read, err
		} else {
			read += n
		}

		headers[i].predefined = EmptyKey
		headers[i].off = uint32(offset)
		offset += int(headers[i].len)

		if headers[i].typ == Type(EndOfObject) {
			break
		}
	}

	headers = headers[:len(headers)-1]

	return headers, read, nil
}

func (vh *ValueHeader) parse(r io.Reader, buf []byte) (int, error) {
	if _, err := io.ReadFull(r, buf[:1]); err != nil {
		if err == io.EOF {
			return 0, ErrTruncatedData
		}
		return 0, err
	}

	if buf[0] == EndOfObject {
		vh.typ = Type(EndOfObject)
		return 1, nil
	}

	read := 1
	switch buf[0] & 0x07 {
	case 0x01:
		vh.typ = TypeNull
		vh.literal = true
		vh.value = nil
	case 0x02:
		vh.typ = TypeBool
		vh.literal = true
		vh.value = buf[0] == LitTrue
	case 0x03:
		vh.typ = TypeInt
		intLen := int(buf[0] >> 3 & 0x07)
		if intLen >= len(intLengthMap) {
			return 1, ErrValueTooLarge
		}
		vh.len = intLengthMap[intLen]

		if vh.len == 0 {
			vh.literal = true
			vh.value = int64(0)
		} else if buf[0]&0x40 != 0 {
			vh.sign = -1
		} else {
			vh.sign = 1
		}
	case 0x04:
		vh.typ = TypeFloat
		vh.len = uint32(buf[0]>>3) & 0x07

		if vh.len == 0 {
			vh.literal = true
			vh.value = float64(0)
		}
	case 0x05:
		vlen, n, err := parseLength(r, buf[0], buf)
		read += n
		if err != nil {
			return read, fmt.Errorf("failed to parse string length: %w", err)
		}
		vh.typ = TypeString
		vh.len = vlen
	case 0x06:
		vlen, n, err := parseLength(r, buf[0], buf)
		read += n
		if err != nil {
			return read, fmt.Errorf("failed to parse object length: %w", err)
		}
		vh.typ = TypeObject
		vh.len = vlen
		if vh.len == 0 {
			vh.literal = true
		}
	case 0x07:
		vlen, n, err := parseLength(r, buf[0], buf)
		read += n
		if err != nil {
			return read, fmt.Errorf("failed to parse array length: %w", err)
		}
		vh.typ = TypeArray
		vh.len = vlen
	default:
		return read, fmt.Errorf("unsupported field type: %v", buf[0]&0x07)
	}

	return read, nil
}

func parseObjectHeader(r io.Reader) ([]ValueHeader, int, error) {
	var headers []ValueHeader
	buf := make([]byte, 4)
	read := 0
	offset := 0

	for {
		if c, err := r.Read(buf[:1]); err != nil {
			return nil, read, fmt.Errorf("failed to read field header: %w", err)
		} else if c == 0 {
			return nil, read, ErrTruncatedData
		} else {
			read += c
		}

		orig := buf[0]

		if buf[0] == EndOfObject {
			break
		}

		fh := ValueHeader{}
		if buf[0]&0x80 != 0 {
			fh.predefined = buf[0] & 0x7F
		} else if buf[0]&0x40 != 0 {
			fh.nameOff = uint32(offset)
			if c, err := r.Read(buf[1:2]); err != nil {
				return nil, read, fmt.Errorf("failed to read extended name length for field: %w", err)
			} else if c == 0 {
				return nil, read, ErrTruncatedData
			} else {
				read += c
			}

			fh.nameLen = (uint32(buf[0]&0x3F) << 8) | uint32(buf[1])
		} else {
			fh.nameOff = uint32(offset)
			fh.nameLen = uint32(buf[0] & 0x3F)
		}

		if fh.nameLen > 1000 {
			log.Printf("Warning: field name length %d exceeds 1000 bytes, which may indicate malformed data", fh.nameLen)
		}

		offset += int(fh.nameLen)

		n, err := fh.parse(r, buf)
		read += n
		if err != nil {
			return nil, read, fmt.Errorf("failed to parse field header: %w", err)
		}
		if fh.typ == Type(EndOfObject) {
			return nil, read, fmt.Errorf("unexpected end of object while parsing field header")
		}

		fh.off = uint32(offset)
		offset += int(fh.len)

		log.Printf("Parsed field header: predefined=%d, nameOff=%d, nameLen=%d, typ=%v, off=%d, len=%d for input byte 0x%02X", fh.predefined, fh.nameOff, fh.nameLen, fh.typ, fh.off, fh.len, orig)
		headers = append(headers, fh)
	}
	return headers, read, nil
}

type objectReader struct {
	dbInfo  *DBInfo
	reader  io.ReadSeeker
	offset  int64
	headers []ValueHeader
}

func ReadObject(dbInfo *DBInfo, reader io.ReadSeeker, offset int64) (ValueCursor, []ValueHeader, error) {
	if _, err := reader.Seek(offset, io.SeekStart); err != nil {
		return nil, nil, err
	}

	headers, hdrlen, err := parseObjectHeader(reader)
	if err != nil {
		return nil, nil, err
	}

	return &objectReader{
		dbInfo:  dbInfo,
		reader:  reader,
		offset:  offset + int64(hdrlen),
		headers: headers,
	}, headers, nil
}

func (or *objectReader) Name(field int) (string, error) {
	fh := or.headers[field]
	if fh.predefined != 0 {
		return or.dbInfo.Key(fh.predefined), nil
	}

	if fh.nameLen == 0 {
		return "", nil
	}

	if _, err := or.reader.Seek(or.offset+int64(fh.nameOff), io.SeekStart); err != nil {
		return "", err
	}

	nameBytes := make([]byte, fh.nameLen)
	if _, err := io.ReadFull(or.reader, nameBytes); err != nil {
		return "", err
	}

	return string(nameBytes), nil
}

func (or *objectReader) Scalar(field int) (ValueParser, error) {
	fh := or.headers[field]
	if !fh.literal {
		if _, err := or.reader.Seek(or.offset+int64(fh.off), io.SeekStart); err != nil {
			return nil, err
		}
	}

	switch fh.typ {
	case TypeNull:
		return nullParser{}, nil
	case TypeBool:
		return boolParser{value: fh.value.(bool)}, nil
	case TypeInt:
		if fh.literal {
			return intParser{}, nil
		}

		ibs := make([]byte, fh.len)
		if _, err := io.ReadFull(or.reader, ibs); err != nil {
			return nil, fmt.Errorf("failed to read int bytes for field %d: %w", field, err)
		}

		return intParser{
			sign:  fh.sign,
			value: ibs,
		}, nil
	case TypeFloat:
		if fh.literal {
			return floatParser{}, nil
		}

		fbs := make([]byte, fh.len)
		if _, err := io.ReadFull(or.reader, fbs); err != nil {
			return nil, fmt.Errorf("failed to read float bytes for field %d: %w", field, err)
		}

		return floatParser{value: fbs}, nil
	case TypeString:
		if fh.literal {
			return stringParser{}, nil
		}

		return &stringParser{
			reader: or.reader,
			length: fh.len,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported field type: %v", fh.typ)
	}
}

func (or *objectReader) ObjectHeader(field int) ([]ValueHeader, error) {
	fh := or.headers[field]
	if fh.typ != TypeObject && fh.typ != TypeArray {
		return nil, fmt.Errorf("unsupported field type: %v", fh.typ)
	}

	if fh.literal {
		return nil, nil
	}

	if fh.hdrLen != 0 {
		return fh.headers, nil
	}

	if _, err := or.reader.Seek(or.offset+int64(fh.off), io.SeekStart); err != nil {
		return nil, err
	}

	var headers []ValueHeader
	var read int
	var err error

	if fh.typ == TypeObject {
		headers, read, err = parseObjectHeader(or.reader)
	} else {
		headers, read, err = parseArrayHeader(or.reader)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse headers for field %d: %w", field, err)
	}

	or.headers[field].hdrLen = uint32(read)
	or.headers[field].headers = headers

	return headers, nil
}

func (or *objectReader) Object(field int) (ValueCursor, error) {
	fh := or.headers[field]
	if fh.typ != TypeObject && fh.typ != TypeArray {
		return nil, fmt.Errorf("unsupported field type: %v", fh.typ)
	}

	if fh.literal {
		return &EmptyFieldCursor{}, nil
	}

	if fh.hdrLen == 0 {
		if _, err := or.ObjectHeader(field); err != nil {
			return nil, err
		}
		fh = or.headers[field]
	}

	return &objectReader{
		dbInfo:  or.dbInfo,
		reader:  or.reader,
		offset:  or.offset + int64(fh.off+fh.hdrLen),
		headers: fh.headers,
	}, nil
}
