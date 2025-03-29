package transport

import (
	"encoding/json"
	"iter"
)

// Discard is a transport that discards all data.
// Discard reads and writes are no-ops.
// This is useful for testing.
type Discard struct{}

func (d Discard) Receive() iter.Seq[json.RawMessage] {
	return func(yield func(json.RawMessage) bool) {}
}
func (d Discard) Send(v json.RawMessage) error {
	return nil
}

func (d Discard) Close() error {
	return nil
}
