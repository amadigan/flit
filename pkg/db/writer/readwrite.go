package writer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/amadigan/flit/internal/util"
	"github.com/amadigan/flit/pkg/db"
	"github.com/google/uuid"
)

// 92fb411e-5da0-4239-b018-dc082b51a426
var ReadWriteFormat = uuid.UUID{0x92, 0xfb, 0x41, 0x1e, 0x5d, 0xa0, 0x42, 0x39, 0xb0, 0x18, 0xdc, 0x8, 0x2b, 0x51, 0xa4, 0x26}

var rwFooterKeys = []string{"database_id", "documents", "start", "length"}
var rwFooterKeyMap = BuildKeyMap(rwFooterKeys)

const rwHeaderLength = 20

func BuildKeyMap(keys []string) map[string]uint8 {
	keyMap := make(map[string]uint8, len(keys))
	for i, key := range keys {
		keyMap[key] = uint8(i + 0x80 + db.BuiltinKeys)
	}
	return keyMap
}

type ReadWriteDB struct {
	file      *os.File
	id        uuid.UUID
	documents map[string]documentRange
	free      util.Ranges
	end       int64
	mutex     sync.RWMutex
}

type documentRange struct {
	Start  int64  `bdoc:"start"`
	Length uint32 `bdoc:"length"`
}

type readWriteFooter struct {
	DatabaseId uuid.UUID                `bdoc:"database_id"`
	Documents  map[string]documentRange `bdoc:"documents"`
}

func OpenReadWriteDB(file *os.File, id uuid.UUID) (*ReadWriteDB, error) {
	database := &ReadWriteDB{
		file: file,
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", file.Name(), err)
	}

	header := make([]byte, rwHeaderLength)

	if stat.Size() == 0 {
		// new file
		copy(header, ReadWriteFormat[:])
		if _, err := file.WriteAt(header, 0); err != nil {
			return nil, fmt.Errorf("failed to write header to file %s: %w", file.Name(), err)
		}

		database.id = id
		database.documents = make(map[string]documentRange)
		database.end = rwHeaderLength
		return database, nil
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start of file %s: %w", file.Name(), err)
	}

	if _, err := io.ReadFull(file, header); err != nil {
		return nil, fmt.Errorf("failed to read header from file %s: %w", file.Name(), err)
	}

	formatId := uuid.UUID{}
	copy(formatId[:], header[:16])
	if formatId != ReadWriteFormat {
		return nil, fmt.Errorf("invalid format: %s", formatId)
	}

	footerLength := binary.BigEndian.Uint32(header[16:20])

	if footerLength == 0 || int64(footerLength) > stat.Size()-rwHeaderLength {
		return nil, fmt.Errorf("invalid footer length: %d", footerLength)
	}

	dbinfo := &db.DBInfo{Keys: rwFooterKeys}
	cursor, headers, err := db.ReadObject(dbinfo, file, stat.Size()-int64(footerLength))
	if err != nil {
		return nil, fmt.Errorf("failed to read footer from file %s: %w", file.Name(), err)
	}

	var footer readWriteFooter
	if err := db.Unmarshal(cursor, headers, &footer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal footer from file %s: %w", file.Name(), err)
	}

	database.id = footer.DatabaseId
	database.documents = footer.Documents
	database.free, database.end = computeFreeRanges(footer.Documents)

	return database, nil
}

func (db *ReadWriteDB) Terminate() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.file.Close()
}

func (db *ReadWriteDB) Close() error {
	log.Printf("Closing database %s with %d documents", db.id, len(db.documents))
	db.mutex.Lock()
	defer db.mutex.Unlock()
	footer := readWriteFooter{
		DatabaseId: db.id,
		Documents:  db.documents,
	}

	fields, err := MarshalDocument(rwFooterKeyMap, footer)
	if err != nil {
		return err
	}

	data, length, err := BuildObject(rwFooterKeyMap, fields)
	if err != nil {
		return err
	}

	if _, err := db.file.Seek(db.end, io.SeekStart); err != nil {
		return err
	}

	for _, part := range data {
		if _, err := db.file.Write(part); err != nil {
			return err
		}
	}

	if err := db.file.Truncate(db.end + int64(length)); err != nil {
		return err
	}

	header := make([]byte, rwHeaderLength)
	copy(header, ReadWriteFormat[:])
	binary.BigEndian.PutUint32(header[16:20], uint32(length))

	if _, err := db.file.WriteAt(header, 0); err != nil {
		return err
	}

	return db.file.Close()
}

func computeFreeRanges(documents map[string]documentRange) (util.Ranges, int64) {
	var free util.Ranges
	var end int64 = rwHeaderLength

	docs := make([]documentRange, 0, len(documents))
	for _, doc := range documents {
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Start < docs[j].Start
	})

	for _, doc := range docs {
		if doc.Start > end {
			free.Add(end, doc.Start)
		}
		end = doc.Start + int64(doc.Length)
	}

	return free, end
}

func (db *ReadWriteDB) DocumentExists(id string) bool {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	_, exists := db.documents[id]
	return exists
}

func (db *ReadWriteDB) OpenDocument(id string) *io.SectionReader {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return db.openDocument(id)
}

func (db *ReadWriteDB) openDocument(id string) *io.SectionReader {
	doc, exists := db.documents[id]
	if !exists {
		return nil
	}

	return io.NewSectionReader(db.file, doc.Start, int64(doc.Length))
}

func (db *ReadWriteDB) deleteDocument(id string) {
	doc, exists := db.documents[id]
	if !exists {
		return
	}

	if db.end == doc.Start+int64(doc.Length) {
		db.end = doc.Start
	} else {
		db.free.Add(doc.Start, doc.Start+int64(doc.Length))
	}

	delete(db.documents, id)
}

func (db *ReadWriteDB) DeleteDocument(id string) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.deleteDocument(id)
}

func (db *ReadWriteDB) WriteDocumentParts(id string, data [][]byte, length int) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if len(data) == 0 {
		return fmt.Errorf("cannot write empty document")
	}

	existing, exists := db.documents[id]
	if exists {
		if existing.Length == uint32(length) {
			old := db.openDocument(id)
			var buf []byte
			mismatched := false
			for _, part := range data {
				if cap(buf) < len(part) {
					buf = make([]byte, 0, len(part))
				}

				if _, err := io.ReadFull(old, buf[:len(part)]); err != nil {
					return err
				}

				if !bytes.Equal(buf[:len(part)], part) {
					mismatched = true
					break
				}
			}

			if !mismatched {
				return nil
			}
		}

		db.deleteDocument(id)
	}

	if db.free.Len() > 0 {
		var smallest util.Subrange
		for r := range db.free.All() {
			if int64(r.End-r.Start) >= int64(length) {
				if smallest.Start == 0 || r.End-r.Start < smallest.End-smallest.Start {
					smallest = r
				}
			}
		}

		if smallest.Start != 0 {
			db.documents[id] = documentRange{
				Start:  smallest.Start,
				Length: uint32(length),
			}

			offset := smallest.Start
			for _, part := range data {
				if _, err := db.file.WriteAt(part, offset); err != nil {
					return err
				}
				offset += int64(len(part))
			}

			db.free.Remove(smallest.Start, smallest.Start+int64(length))
			return nil
		}
	}

	db.documents[id] = documentRange{
		Start:  db.end,
		Length: uint32(length),
	}

	offset := db.end
	for _, part := range data {
		if _, err := db.file.WriteAt(part, offset); err != nil {
			return err
		}
		offset += int64(len(part))
	}

	db.end += int64(length)
	return nil
}

func (db *ReadWriteDB) GetDocumentIds() []string {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	ids := make([]string, 0, len(db.documents))
	for id := range db.documents {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids
}

func (db *ReadWriteDB) GetDocumentCount() int {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return len(db.documents)
}

func (rdb *ReadWriteDB) UnmarshalDocument(dbinfo *db.DBInfo, id string, v any) error {
	reader := rdb.OpenDocument(id)
	if reader == nil {
		return fmt.Errorf("document %q not found", id)
	}

	cursor, headers, err := db.ReadObject(dbinfo, reader, 0)
	if err != nil {
		return err
	}

	return db.Unmarshal(cursor, headers, v, db.ExtraField{Name: dbinfo.IdKey, Value: db.StaticString(id)})
}
