package transport

import "io"

// NewPipe creates a pair of transports that are connected to each other.
// This is useful for testing.
func NewPipe() (a, b Session) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	return NewGeneric(r1, w2), NewGeneric(r2, w1)
}
