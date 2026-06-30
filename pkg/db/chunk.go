package db

import "io"

type ChunkHeader struct {
	DatabaseId []byte
	Chunk      uint16
	LastChunk  bool
	FirstDoc   uint32
	Docs       []uint32
}

type ChunkMap struct {
	idOffset uint32
	docs     []uint32
}

type ChunkReader struct {
	reader    io.ReadSeekCloser
	chunkMap  ChunkMap
	dbInfo    *DBInfo
	docOffset int64
	headers   []ValueHeader
	hdrLen    int64
}

func (cr *ChunkReader) Close() error {
	return cr.reader.Close()
}

func (cr *ChunkReader) Next(docId DocumentId) ([]ValueHeader, error) {
	if int(docId) < 0 || int(docId) >= len(cr.chunkMap.docs) {
		return nil, ErrInvalidDocumentId
	}

	cr.docOffset = int64(cr.chunkMap.docs[docId])
	if cr.docOffset == 0 {
		return nil, ErrDocumentNotFound
	}

	if _, err := cr.reader.Seek(cr.docOffset, io.SeekStart); err != nil {
		return nil, err
	}

	headers, hdrlen, err := parseObjectHeader(cr.reader)
	if err != nil {
		return nil, err
	}

	cr.hdrLen = int64(hdrlen)
	cr.headers = headers

	return headers, nil
}

func (cr *ChunkReader) Open() (ValueCursor, error) {
	return &objectReader{
		dbInfo:  cr.dbInfo,
		reader:  cr.reader,
		offset:  cr.docOffset + cr.hdrLen,
		headers: cr.headers,
	}, nil
}

func (cr *ChunkReader) Name(fh ValueHeader) (string, error) {
	if fh.predefined != 0 {
		return cr.dbInfo.Key(fh.predefined), nil
	}

	if fh.nameLen == 0 {
		return "", nil
	}

	if _, err := cr.reader.Seek(cr.docOffset+cr.hdrLen+int64(fh.nameOff), io.SeekStart); err != nil {
		return "", err
	}

	nameBytes := make([]byte, fh.nameLen)
	if _, err := io.ReadFull(cr.reader, nameBytes); err != nil {
		return "", err
	}

	return string(nameBytes), nil
}
