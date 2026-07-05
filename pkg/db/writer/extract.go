package writer

import (
	"strings"
	"sync"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/extractor"
)

type ExtractDBWriter struct {
	db      *ReadWriteDB
	dbinfo  *db.DBInfo
	root    extractor.RootDocument
	end     extractor.EndDocument
	keymap  map[string]uint8
	wg      sync.WaitGroup
	errChan chan<- extractor.WriterError
	done    chan struct{}
	mutex   sync.RWMutex
	sink    chan<- sinkDocument
}

type sinkDocument struct {
	id     string
	root   string
	data   [][]byte
	doc    extractor.Document
	length int
}

func DefaultDBInfo() *db.DBInfo {
	return &db.DBInfo{
		Keys: []string{"id", "type", "order", "parent", "container", "sourceId", "line", "warnings", "errors", "version",
			"lang", "code", "fields", "types", "endLine", "endColumn", "offset", "length"},
		IdKey: "id",
	}
}

func BuildDBInfo(root extractor.RootDocument) *db.DBInfo {
	keys := []string{"id", "type", "order", "parent", "container", "sourceId", "line", "version",
		"lang", "code", "content", "endLine", "endColumn", "offset", "length"}

	keySet := make(map[string]struct{})
	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	for field := range root.Fields {
		field = strings.ToLower(field)
		if _, exists := keySet[field]; !exists {
			keys = append(keys, field)
			keySet[field] = struct{}{}
		}
	}

	return &db.DBInfo{
		Keys:  keys,
		IdKey: "id",
	}
}

func NewExtractDBWriter(db *ReadWriteDB, root extractor.RootDocument, source <-chan extractor.ExtractStream, errChan chan<- extractor.WriterError) *ExtractDBWriter {
	sink := make(chan sinkDocument, 16)

	dbinfo := DefaultDBInfo()

	writer := &ExtractDBWriter{
		db:      db,
		dbinfo:  dbinfo,
		root:    root,
		keymap:  BuildKeyMap(dbinfo.Keys),
		errChan: errChan,
		sink:    sink,
		done:    make(chan struct{}),
	}

	go func() {
		for stream := range source {

			writer.wg.Add(1)
			go func(source <-chan extractor.Document) {
				defer writer.wg.Done()
				for doc := range source {
					if err := writer.marshal(doc); err != nil {
						writer.errChan <- extractor.WriterError{
							Document: doc,
							Cause:    err,
						}
					}
				}
			}(stream.Source)
		}
	}()

	go func() {
		defer close(writer.done)

		for doc := range sink {
			if err := writer.db.WriteDocumentParts(doc.id, doc.data, doc.length); err != nil {
				writer.errChan <- extractor.WriterError{
					Document: doc.doc,
					Cause:    err,
				}
			}
		}
	}()

	return writer
}

func (w *ExtractDBWriter) marshal(doc extractor.Document) error {
	fields, err := MarshalDocument(w.keymap, doc)
	if err != nil {
		return err
	}

	data, length, err := BuildObject(w.keymap, fields)
	if err != nil {
		return err
	}

	w.sink <- sinkDocument{
		id:     doc.DocFields().Id,
		data:   data,
		doc:    doc,
		length: length,
	}

	return nil
}
func (w *ExtractDBWriter) CloseRoot(end extractor.EndDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.wg.Wait()
	w.end = end
	return nil
}

func (w *ExtractDBWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.wg.Wait()
	close(w.sink)
	<-w.done

	return nil
}
