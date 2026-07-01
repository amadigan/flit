package writer

import (
	"sync"

	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/extractor"
)

type ExtractDBWriter struct {
	db      *ReadWriteDB
	ends    map[string]extractor.EndDocument
	keymap  map[string]uint8
	wgs     map[string]*sync.WaitGroup
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

func NewExtractDBWriter(db *ReadWriteDB, dbinfo *db.DBInfo, source <-chan extractor.ExtractStream, errChan chan<- extractor.WriterError) *ExtractDBWriter {
	sink := make(chan sinkDocument, 16)
	writer := &ExtractDBWriter{
		db:      db,
		keymap:  BuildKeyMap(dbinfo.Keys),
		wgs:     make(map[string]*sync.WaitGroup),
		errChan: errChan,
		sink:    sink,
		done:    make(chan struct{}),
	}

	go func() {
		for stream := range source {
			writer.mutex.RLock()
			wg, ok := writer.wgs[stream.RootId]
			writer.mutex.RUnlock()

			if !ok {
				writer.errChan <- extractor.WriterError{
					RootId: stream.RootId,
					Cause:  &extractor.RootClosedError{RootId: stream.RootId},
				}
				continue
			}

			wg.Add(1)
			go func(rootId string, source <-chan extractor.Document) {
				defer wg.Done()
				for doc := range source {
					if err := writer.marshal(rootId, doc); err != nil {
						writer.errChan <- extractor.WriterError{
							RootId:   rootId,
							Document: doc,
							Cause:    err,
						}
					}
				}
			}(stream.RootId, stream.Source)
		}
	}()

	go func() {
		defer close(writer.done)

		for doc := range sink {
			if err := writer.db.WriteDocumentParts(doc.id, doc.data, doc.length); err != nil {
				writer.errChan <- extractor.WriterError{
					RootId:   doc.root,
					Document: doc.doc,
					Cause:    err,
				}
			}
		}
	}()

	return writer
}

func (w *ExtractDBWriter) marshal(root string, doc extractor.Document) error {
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
		root:   root,
		data:   data,
		doc:    doc,
		length: length,
	}

	return nil
}

func (w *ExtractDBWriter) Open(root extractor.RootDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, ok := w.wgs[root.Id]; !ok {
		w.wgs[root.Id] = &sync.WaitGroup{}
	}

	return w.marshal(root.Id, root)
}

func (w *ExtractDBWriter) WaitForRoot(rootId string) {
	w.mutex.RLock()
	wg, ok := w.wgs[rootId]
	w.mutex.RUnlock()

	if !ok {
		return
	}

	wg.Wait()
}

func (w *ExtractDBWriter) CloseRoot(end extractor.EndDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if wg, ok := w.wgs[end.Id]; !ok {
		return &extractor.RootClosedError{RootId: end.Id}
	} else {
		wg.Wait()
		delete(w.wgs, end.Id)
	}

	if w.ends == nil {
		w.ends = make(map[string]extractor.EndDocument)
	}

	w.ends[end.Id] = end

	return nil
}

func (w *ExtractDBWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for _, wg := range w.wgs {
		wg.Wait()
	}
	close(w.sink)
	<-w.done

	return nil
}
