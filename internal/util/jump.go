package util

import (
	"errors"
	"io"
)

type jumpReader struct {
	reader io.Reader
	offset int64
	length int64
}

var ErrorReverseSeek = errors.New("reverse seek not supported")

func (jr *jumpReader) Read(p []byte) (int, error) {
	n, err := jr.reader.Read(p)
	jr.offset += int64(n)
	return n, err
}

func (jr *jumpReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = jr.offset + offset
	case io.SeekEnd:
		newOffset = jr.length + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newOffset < jr.offset {
		return 0, ErrorReverseSeek
	}

	if newOffset > jr.offset {
		buf := make([]byte, min(newOffset-jr.offset, 4096))
		for newOffset > jr.offset {
			if newOffset-jr.offset < int64(len(buf)) {
				buf = buf[:newOffset-jr.offset]
			}
			n, err := jr.Read(buf)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				return 0, io.EOF
			}
		}
	}

	return newOffset, nil
}
