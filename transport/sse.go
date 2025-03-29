package transport

import (
	"bytes"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
)

type SSE struct {
	// baseURL is the baseURL for the SSE session handler.
	baseURL string
	// basePath is the basePath for the SSE session handler path.
	basePath string
	// handler is the handler for the SSE session.
	handler SessionHandler

	idSampler *rand.ChaCha8
	mu        sync.Mutex
	sessions  map[uint64]*SSESession
}

func NewSSE(prefix string, handler SessionHandler) (*SSE, error) {
	var baseURL, basePath string
	if prefix != "" {
		u, err := url.Parse(prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid prefix: %w", err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("scheme must be http or https: %s", u.Scheme)
		}

		if u.Host == "" {
			return nil, fmt.Errorf("host is required")
		}

		if len(u.Query()) != 0 {
			return nil, fmt.Errorf("query is not allowed: %s", u.Query())
		}

		if u.Fragment != "" {
			return nil, fmt.Errorf("fragment is not allowed: %s", u.Fragment)
		}

		baseURL = strings.TrimSuffix(u.String(), u.Path)
		basePath = strings.TrimSuffix(u.Path, "/")
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
	}

	var seed [32]byte
	if _, err := crand.Read(seed[:]); err != nil {
		return nil, fmt.Errorf("failed to generate seed: %w", err)
	}

	return &SSE{
		baseURL:   baseURL,
		basePath:  basePath,
		handler:   handler,
		idSampler: rand.NewChaCha8(seed),
		sessions:  make(map[uint64]*SSESession),
	}, nil
}

func (s *SSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == s.basePath || strings.TrimSuffix(r.URL.Path, "/") == s.basePath {
		s.handleSSE(w, r)
		return
	}

	s.handleMessage(w, r)
}

func (s *SSE) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(interface {
		io.Writer
		http.Flusher
	})
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	session := &SSESession{
		ch:     make(chan io.Reader),
		writer: flusher,
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id := s.idSampler.Uint64()

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	p, err := url.JoinPath(s.baseURL, s.basePath, strconv.FormatUint(id, 10))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", p)
	flusher.Flush()

	if err := s.handler.HandleSession(r.Context(), id, session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	select {
	case <-session.done:
	case <-r.Context().Done():
		close(session.ch)
	}

	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *SSE) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p, err := url.JoinPath(s.basePath, path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Path != p {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	s.mu.Lock()
	session, ok := s.sessions[id]
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.ch <- bytes.NewReader(b)
}

type SSESession struct {
	ch     chan io.Reader
	writer interface {
		io.Writer
		http.Flusher
	}
	done chan struct{}
}

var ErrSessionClosed = errors.New("session closed")

func (s *SSESession) Send(v json.RawMessage) error {
	select {
	case <-s.done:
		return ErrSessionClosed
	default:
		if _, err := fmt.Fprintf(s.writer, "event: message\ndata: %s\n\n", string(v)); err != nil {
			return errors.Join(err, s.Close())
		}
		s.writer.Flush()
		return nil
	}
}

func (s *SSESession) Receive() iter.Seq[json.RawMessage] {
	return func(yield func(json.RawMessage) bool) {
		for r := range s.ch {
			d := json.NewDecoder(r)
			for {
				var v json.RawMessage
				if err := d.Decode(&v); err != nil {
					if !errors.Is(err, io.EOF) {
						return
					}
					break
				}
				if !yield(v) {
					return
				}
			}
		}
	}
}

func (s *SSESession) Close() error {
	select {
	case <-s.done:
		return nil
	default:
		close(s.done)
		close(s.ch)
	}
	return nil
}
