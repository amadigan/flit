package extractor

import (
	"encoding/json"
	"io"
	"sync"
)

type ExtractWriter struct {
	wg      sync.WaitGroup
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
	Document Document
	Cause    error
}

func (e *WriterError) Error() string {
	return "error writing document: " + e.Cause.Error()
}

func (e *WriterError) Unwrap() error {
	return e.Cause
}

type ExtractStream struct {
	Source <-chan Document
}

func NewExtractWriter(source <-chan ExtractStream, errChan chan<- WriterError, out io.Writer) *ExtractWriter {
	sink := make(chan []byte, 16)
	writer := &ExtractWriter{
		errChan: errChan,
		sink:    sink,
		done:    make(chan struct{}),
	}

	go func() {
		for stream := range source {
			writer.wg.Add(1)
			go func(stream ExtractStream) {
				defer writer.wg.Done()
				for doc := range stream.Source {
					data, err := json.Marshal(doc)
					if err != nil {
						writer.errChan <- WriterError{
							Document: doc,
							Cause:    err,
						}
						continue
					}

					sink <- append(data, '\n')
				}
			}(stream)
		}

		writer.wg.Wait()
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

	bs, err := json.Marshal(root)
	if err != nil {
		return err
	}

	w.sink <- append(bs, '\n')
	return nil
}

func (w *ExtractWriter) CloseRoot(end EndDocument) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.wg.Wait()

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

	w.wg.Wait()

	close(w.sink)
	<-w.done

	return nil
}
