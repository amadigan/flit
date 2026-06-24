package extractor

import (
	"encoding/json"
	"io"
	"sync"
)

type ExtractWriter struct {
	wgs     map[string]*sync.WaitGroup
	errChan chan<- WriterError
	done    chan struct{}
	mutex   sync.RWMutex
	sink    chan<- []byte
}

type RootClosedError struct {
	RootId string
}

func (e *RootClosedError) Error() string {
	return "root closed: " + e.RootId
}

type WriterError struct {
	RootId   string
	Document Document
	Cause    error
}

func (e *WriterError) Error() string {
	return "error writing document for root " + e.RootId + ": " + e.Cause.Error()
}

func (e *WriterError) Unwrap() error {
	return e.Cause
}

type ExtractStream struct {
	RootId string
	Source <-chan Document
}

func NewExtractWriter(source <-chan ExtractStream, errChan chan<- WriterError, out io.Writer) *ExtractWriter {
	sink := make(chan []byte, 16)
	writer := &ExtractWriter{
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
				writer.errChan <- WriterError{
					RootId: stream.RootId,
					Cause:  &RootClosedError{RootId: stream.RootId},
				}
				continue
			}

			wg.Add(1)
			go func(stream ExtractStream, wg *sync.WaitGroup) {
				defer wg.Done()
				for doc := range stream.Source {
					data, err := json.Marshal(doc)
					if err != nil {
						writer.errChan <- WriterError{
							RootId:   stream.RootId,
							Document: doc,
							Cause:    err,
						}
						continue
					}

					sink <- append(data, '\n')
				}
			}(stream, wg)
		}

		writer.mutex.Lock()
		for _, wg := range writer.wgs {
			wg.Wait()
		}
		writer.mutex.Unlock()
		close(sink)
		close(writer.errChan)
	}()

	go func() {
		defer close(writer.done)
		for data := range sink {
			out.Write(data)
		}
	}()

	return writer
}

func (w *ExtractWriter) Open(root RootDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, ok := w.wgs[root.Id]; !ok {
		w.wgs[root.Id] = &sync.WaitGroup{}
	}

	bs, err := json.Marshal(root)
	if err != nil {
		return err
	}

	w.sink <- append(bs, '\n')
	return nil
}

func (w *ExtractWriter) WaitForRoot(rootId string) {
	w.mutex.RLock()
	wg, ok := w.wgs[rootId]
	w.mutex.RUnlock()

	if !ok {
		return
	}

	wg.Wait()
}

func (w *ExtractWriter) CloseRoot(end EndDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	wg, ok := w.wgs[end.Id]
	if !ok {
		return &RootClosedError{RootId: end.Id}
	}

	wg.Wait()
	delete(w.wgs, end.Id)

	bs, err := json.Marshal(end)
	if err != nil {
		return err
	}

	w.sink <- append(bs, '\n')
	return nil
}

func (w *ExtractWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for _, wg := range w.wgs {
		wg.Wait()
	}
	delete(w.wgs, "")

	close(w.sink)
	<-w.done

	return nil
}
