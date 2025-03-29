package transport

import (
	"io"
	"net/http"
)

// BaseURL returns the base URL configured for the SSE handler.
// This is exported for testing purposes.
func (s *SSE) BaseURL() string {
	return s.baseURL
}

// BasePath returns the base path configured for the SSE handler.
// This is exported for testing purposes.
func (s *SSE) BasePath() string {
	return s.basePath
}

// SessionCount returns the current number of active SSE sessions.
// This is exported for testing purposes.
func (s *SSE) SessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// HasSession checks if a session with the given ID exists.
// This is exported for testing purposes.
func (s *SSE) HasSession(id uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[id]
	return ok
}

// NewSSESession creates a new SSE session with the given writer.
// This is exported for testing purposes.
func NewSSESession(writer interface {
	io.Writer
	http.Flusher
}) *SSESession {
	return &SSESession{
		writer: writer,
		ch:     make(chan io.Reader),
		done:   make(chan struct{}),
	}
}

// HandleMessage handles a message from the SSE session.
// This is exported for testing purposes.
func (s *SSESession) HandleMessage(r io.Reader) {
	s.ch <- r
}
