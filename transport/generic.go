package transport

import (
	"encoding/json"
	"errors"
	"io"
	"iter"
	"os"
)

// Generic is a transport for the Model Context Protocol that uses the standard input and output.
type Generic struct {
	reader io.Reader
	writer io.Writer
}

// NewGeneric creates a new Transport that uses the standard input and output.
func NewGeneric(r io.Reader, w io.Writer) *Generic {
	return &Generic{
		reader: r,
		writer: w,
	}
}

// NewStdio creates a new Transport that uses the standard input and output.
func NewStdio() *Generic {
	return NewGeneric(os.Stdin, os.Stdout)
}

// Receive reads JSON messages from the reader.
func (t *Generic) Receive() iter.Seq[json.RawMessage] {
	jsonReader := json.NewDecoder(t.reader)
	return func(yield func(json.RawMessage) bool) {
		for {
			var v json.RawMessage
			if err := jsonReader.Decode(&v); err != nil {
				return
			}
			if !yield(v) {
				return
			}
		}
	}
}

// Send writes a JSON message to the writer, followed by a newline.
func (t *Generic) Send(v json.RawMessage) error {
	_, err := t.writer.Write(v)
	if err != nil {
		return err
	}
	_, err = t.writer.Write([]byte("\n"))
	return err
}

// Close closes the transport's reader and writer if they implement io.Closer.
// It avoids closing the same resource twice if reader and writer are identical.
func (t *Generic) Close() error {
	var rErr, wErr error
	closedReader := false

	if rCloser, ok := t.reader.(io.Closer); ok {
		rErr = rCloser.Close()
		closedReader = true
	}

	if wCloser, ok := t.writer.(io.Closer); ok {
		shouldCloseWriter := true
		if closedReader {
			if rCloser, rOk := t.reader.(io.Closer); rOk && rCloser == wCloser {
				shouldCloseWriter = false
			}
		}

		if shouldCloseWriter {
			wErr = wCloser.Close()
		}
	}

	return errors.Join(rErr, wErr)
}
