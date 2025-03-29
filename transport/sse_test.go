package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type mockSessionHandler struct {
	HandleSessionFunc func(context.Context, Session) (uint64, error)
}

func (m *mockSessionHandler) HandleSession(ctx context.Context, s Session) (uint64, error) {
	if m.HandleSessionFunc == nil {
		return 0, nil
	}
	return m.HandleSessionFunc(ctx, s)
}

type mockFlusher struct {
	*bytes.Buffer
	flushed bool
}

func (m *mockFlusher) Flush() {
	m.flushed = true
}

func (m *mockFlusher) Header() http.Header {
	return http.Header{}
}

func (m *mockFlusher) WriteHeader(statusCode int) {}

func TestNewSSE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		prefix       string
		wantBaseURL  string
		wantBasePath string
		wantErr      bool
	}{
		{
			name:         "empty prefix",
			prefix:       "",
			wantBaseURL:  "",
			wantBasePath: "",
			wantErr:      false,
		},
		{
			name:         "valid http prefix",
			prefix:       "http://localhost:8080/sse",
			wantBaseURL:  "http://localhost:8080",
			wantBasePath: "/sse",
			wantErr:      false,
		},
		{
			name:         "valid https prefix with trailing slash",
			prefix:       "https://example.com/api/",
			wantBaseURL:  "https://example.com",
			wantBasePath: "/api",
			wantErr:      false,
		},
		{
			name:         "valid prefix root path",
			prefix:       "http://localhost:8080",
			wantBaseURL:  "http://localhost:8080",
			wantBasePath: "",
			wantErr:      false,
		},
		{
			name:    "invalid prefix format",
			prefix:  "://invalid",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			prefix:  "ftp://localhost/sse",
			wantErr: true,
		},
		{
			name:    "missing host",
			prefix:  "http:///sse",
			wantErr: true,
		},
		{
			name:    "prefix with query",
			prefix:  "http://localhost/sse?query=param",
			wantErr: true,
		},
		{
			name:    "prefix with fragment",
			prefix:  "http://localhost/sse#fragment",
			wantErr: true,
		},
	}

	handler := &mockSessionHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewSSE(tt.prefix, handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSSE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Errorf("NewSSE() got = nil, want non-nil")
					return
				}
				if got.BaseURL() != tt.wantBaseURL {
					t.Errorf("NewSSE() BaseURL = %q, want %q", got.BaseURL(), tt.wantBaseURL)
				}
				if got.BasePath() != tt.wantBasePath {
					t.Errorf("NewSSE() BasePath = %q, want %q", got.BasePath(), tt.wantBasePath)
				}
			}
		})
	}
}

func TestSSESession_Send(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   json.RawMessage
		want    string
		wantErr bool
	}{
		{
			name:  "simple message",
			input: json.RawMessage(`{"hello":"world"}`),
			want:  "event: message\ndata: {\"hello\":\"world\"}\n\n",
		},
		{
			name:  "empty object",
			input: json.RawMessage(`{}`),
			want:  "event: message\ndata: {}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := &mockFlusher{Buffer: &bytes.Buffer{}}
			s := NewSSESession(buf)

			err := s.Send(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSESession.Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got := buf.String(); got != tt.want {
				t.Errorf("SSESession.Send() = %v, want %v", got, tt.want)
			}

			if !buf.flushed {
				t.Error("SSESession.Send() did not flush the writer")
			}
		})
	}
}

func TestSSESession_Receive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		messages []json.RawMessage
		want     []json.RawMessage
	}{
		{
			name: "single message",
			messages: []json.RawMessage{
				json.RawMessage(`{"hello":"world"}`),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"hello":"world"}`),
			},
		},
		{
			name: "multiple messages",
			messages: []json.RawMessage{
				json.RawMessage(`{"hello":"world"}`),
				json.RawMessage(`{"foo":"bar"}`),
			},
			want: []json.RawMessage{
				json.RawMessage(`{"hello":"world"}`),
				json.RawMessage(`{"foo":"bar"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewSSESession(&mockFlusher{Buffer: &bytes.Buffer{}})

			// Send messages through a channel
			go func() {
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				for _, msg := range tt.messages {
					enc.Encode(msg)
				}
				s.HandleMessage(&buf)
				s.Close()
			}()

			var got []json.RawMessage
			for v := range s.Receive() {
				got = append(got, v)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("SSESession.Receive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSESession_Close(t *testing.T) {
	t.Parallel()
	s := NewSSESession(&mockFlusher{Buffer: &bytes.Buffer{}})

	if err := s.Close(); err != nil {
		t.Errorf("SSESession.Close() error = %v", err)
	}

	// Test that calling Close() multiple times is safe
	if err := s.Close(); err != nil {
		t.Errorf("SSESession.Close() second call error = %v", err)
	}

	// Verify that after closing, Send returns an error
	if err := s.Send(json.RawMessage(`{}`)); err == nil {
		t.Error("SSESession.Send() after Close() should return an error")
	}

	// Verify that after closing, Receive channel is closed
	for range s.Receive() {
		t.Error("SSESession.Receive() after Close() should be closed")
	}
}

// testResponseRecorder wraps httptest.ResponseRecorder and implements http.Flusher.
type testResponseRecorder struct {
	*httptest.ResponseRecorder
}

// Flush implements http.Flusher. No-op.
func (r *testResponseRecorder) Flush() {}

// testSessionHandler implements SessionHandler for testing SSE handshake.
// It sets a closed done channel to prevent blocking in handleSSE.
type testSessionHandler struct{}

func (h *testSessionHandler) HandleSession(ctx context.Context, s Session) (uint64, error) {
	// Assert that s is of type *SSESession
	ssession, ok := s.(*SSESession)
	if !ok {
		return 0, errors.New("invalid session type")
	}
	// Create and immediately close the done channel to unblock select in handleSSE.
	ch := make(chan struct{})
	close(ch)
	ssession.done = ch
	return 42, nil
}

// dummyFlusher implements a minimal writer and Flush for testing message handler.
type dummyFlusher struct{}

func (d dummyFlusher) Write(p []byte) (int, error) { return len(p), nil }
func (d dummyFlusher) Flush()                      {}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		method                string
		path                  string
		body                  string
		setup                 func(sse *SSE)
		expectedStatus        int
		expectedBodySubstring string
		check                 func(t *testing.T, sse *SSE, rec *testResponseRecorder)
	}{
		{
			name:                  "SSE GET success",
			method:                "GET",
			path:                  "/sse",
			expectedStatus:        http.StatusOK,
			expectedBodySubstring: "event: endpoint",
			setup:                 nil,
			check: func(t *testing.T, sse *SSE, rec *testResponseRecorder) {
				if len(sse.sessions) != 0 {
					t.Errorf("expected sessions map to be empty, got %d", len(sse.sessions))
				}
			},
		},
		{
			name:                  "SSE GET success with trailing slash",
			method:                "GET",
			path:                  "/sse/",
			expectedStatus:        http.StatusOK,
			expectedBodySubstring: "event: endpoint",
			setup:                 nil,
			check: func(t *testing.T, sse *SSE, rec *testResponseRecorder) {
				if len(sse.sessions) != 0 {
					t.Errorf("expected sessions map to be empty, got %d", len(sse.sessions))
				}
			},
		},
		{
			name:                  "SSE wrong method on SSE path",
			method:                "POST",
			path:                  "/sse",
			expectedStatus:        http.StatusMethodNotAllowed,
			expectedBodySubstring: "Method not allowed",
			setup:                 nil,
		},
		{
			name:                  "Message GET wrong method",
			method:                "GET",
			path:                  "/sse/42",
			expectedStatus:        http.StatusMethodNotAllowed,
			expectedBodySubstring: "Method not allowed",
			setup:                 nil,
		},
		{
			name:                  "Message POST with invalid URL pattern",
			method:                "POST",
			path:                  "/sse/extra/42",
			expectedStatus:        http.StatusNotFound,
			expectedBodySubstring: "Not found",
			setup:                 nil,
		},
		{
			name:                  "Message POST session not found",
			method:                "POST",
			path:                  "/sse/42",
			expectedStatus:        http.StatusNotFound,
			expectedBodySubstring: "Session not found",
			setup:                 nil,
		},
		{
			name:                  "Message POST success",
			method:                "POST",
			path:                  "/sse/42",
			body:                  "test message",
			expectedStatus:        http.StatusOK,
			expectedBodySubstring: "",
			setup: func(sse *SSE) {
				// Prepopulate the sessions map with a dummy session for id 42.
				sse.sessions[42] = &SSESession{
					ch:     make(chan io.Reader, 1),
					writer: dummyFlusher{},
					done:   make(chan struct{}),
				}
			},
			check: func(t *testing.T, sse *SSE, rec *testResponseRecorder) {
				sess, exists := sse.sessions[42]
				if !exists {
					t.Fatalf("expected session with id 42 to exist")
				}
				select {
				case r := <-sess.ch:
					data, err := io.ReadAll(r)
					if err != nil {
						t.Fatalf("failed to read from session channel: %v", err)
					}
					if string(data) != "test message" {
						t.Errorf("expected message %q, got %q", "test message", string(data))
					}
				default:
					t.Errorf("expected a message in session channel but got none")
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a new SSE instance for each test case to avoid shared state.
			handler := &testSessionHandler{}
			sse, err := NewSSE("http://localhost/sse", handler)
			if err != nil {
				t.Fatalf("failed to create SSE: %v", err)
			}
			// Initialize the sessions map.
			sse.sessions = make(map[uint64]*SSESession)
			if tc.setup != nil {
				tc.setup(sse)
			}

			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := &testResponseRecorder{httptest.NewRecorder()}

			sse.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
			if tc.expectedBodySubstring != "" && !strings.Contains(rec.Body.String(), tc.expectedBodySubstring) {
				t.Errorf("expected response body to contain %q, got %q", tc.expectedBodySubstring, rec.Body.String())
			}

			if tc.check != nil {
				tc.check(t, sse, rec)
			}
		})
	}
}
