package transport_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/transport"
)

func TestStdio_Send(t *testing.T) {
	t.Parallel()
	var writer bytes.Buffer
	stdio := transport.NewGeneric(nil, &writer) // reader is not used in Send

	testMessage := json.RawMessage(`{"hello":"world"}`)
	err := stdio.Send(testMessage)
	if err != nil {
		t.Fatalf("Send() error = %v, wantErr %v", err, false)
	}

	expectedOutput := string(testMessage) + "\n"
	if writer.String() != expectedOutput {
		t.Errorf("Send() output = %q, want %q", writer.String(), expectedOutput)
	}
}

func TestStdio_Receive(t *testing.T) {
	t.Parallel()
	inputMessages := []json.RawMessage{
		json.RawMessage(`{"message": 1}`),
		json.RawMessage(`{"message": 2}`),
		json.RawMessage(`{"message": 3}`),
	}
	var inputBuilder strings.Builder
	for _, msg := range inputMessages {
		inputBuilder.Write(msg)
		inputBuilder.WriteString("\n")
	}
	reader := strings.NewReader(inputBuilder.String())

	stdio := transport.NewGeneric(reader, nil) // writer is not used in Receive

	receivedMessages := []json.RawMessage{}
	for msg := range stdio.Receive() {
		receivedMessages = append(receivedMessages, msg)
	}

	if len(receivedMessages) != len(inputMessages) {
		t.Fatalf("Receive() received %d messages, want %d", len(receivedMessages), len(inputMessages))
	}
	for i, expected := range inputMessages {
		if string(receivedMessages[i]) != string(expected) {
			t.Errorf("Receive()[%d] = %s, want %s", i, string(receivedMessages[i]), string(expected))
		}
	}
}

func TestStdio_Receive_EOF(t *testing.T) {
	t.Parallel()
	reader := strings.NewReader("") // Empty input
	stdio := transport.NewGeneric(reader, nil)

	count := 0
	for range stdio.Receive() {
		count++
	}

	if count != 0 {
		t.Errorf("Receive() count = %d, want 0 on empty input", count)
	}
}

func TestStdio_Receive_InvalidJSON(t *testing.T) {
	t.Parallel()
	input := `{"valid": 1}` + "\n" + `invalid json` + "\n" + `{"valid": 2}` + "\n"
	reader := strings.NewReader(input)
	stdio := transport.NewGeneric(reader, nil)

	receivedMessages := []json.RawMessage{}
	for msg := range stdio.Receive() {
		receivedMessages = append(receivedMessages, msg)
	}

	// Only the first valid message should be received before the decode error stops iteration.
	expectedCount := 1
	if len(receivedMessages) != expectedCount {
		t.Fatalf("Receive() received %d messages, want %d with invalid JSON input", len(receivedMessages), expectedCount)
	}
	expectedJSON := `{"valid": 1}`
	if string(receivedMessages[0]) != expectedJSON {
		t.Errorf("Receive()[0] = %s, want %s", string(receivedMessages[0]), expectedJSON)
	}
}

// --- Mock Closer Implementations ---

type mockCloser struct {
	io.Reader // Embed Reader or Writer interface
	io.Writer
	closeCalls atomic.Int32
	closeErr   error // Optional error to return on Close
}

func (m *mockCloser) Close() error {
	m.closeCalls.Add(1)
	return m.closeErr
}

func (m *mockCloser) getCloseCalls() int {
	return int(m.closeCalls.Load())
}

// newMockReadCloser creates a mock implementing io.ReadCloser
func newMockReadCloser(r io.Reader, closeErr error) *mockCloser {
	return &mockCloser{Reader: r, closeErr: closeErr}
}

// newMockWriteCloser creates a mock implementing io.WriteCloser
func newMockWriteCloser(w io.Writer, closeErr error) *mockCloser {
	return &mockCloser{Writer: w, closeErr: closeErr}
}

// newMockReadWriteCloser creates a mock implementing io.ReadWriteCloser
func newMockReadWriteCloser(r io.Reader, w io.Writer, closeErr error) *mockCloser {
	return &mockCloser{Reader: r, Writer: w, closeErr: closeErr}
}

func TestStdio_Close_SeparateClosers(t *testing.T) {
	t.Parallel()
	readerMock := newMockReadCloser(strings.NewReader(""), nil)
	writerMock := newMockWriteCloser(&bytes.Buffer{}, nil)

	stdio := transport.NewGeneric(readerMock, writerMock)
	err := stdio.Close()

	if err != nil {
		t.Fatalf("Close() error = %v, wantErr %v", err, false)
	}
	if readerMock.getCloseCalls() != 1 {
		t.Errorf("Reader Close calls = %d, want 1", readerMock.getCloseCalls())
	}
	if writerMock.getCloseCalls() != 1 {
		t.Errorf("Writer Close calls = %d, want 1", writerMock.getCloseCalls())
	}
}

func TestStdio_Close_SameInstance(t *testing.T) {
	t.Parallel()
	// Use a mock that implements both ReadCloser and WriteCloser
	readWriteMock := newMockReadWriteCloser(strings.NewReader(""), &bytes.Buffer{}, nil)

	stdio := transport.NewGeneric(readWriteMock, readWriteMock)
	err := stdio.Close()

	if err != nil {
		t.Fatalf("Close() error = %v, wantErr %v", err, false)
	}
	// Should only call Close once even if it's used for both reader and writer
	if readWriteMock.getCloseCalls() != 1 {
		t.Errorf("Close calls = %d, want 1 on shared instance", readWriteMock.getCloseCalls())
	}
}

func TestStdio_Close_NotCloser(t *testing.T) {
	t.Parallel()
	reader := strings.NewReader("test")
	var writer bytes.Buffer

	// strings.Reader and bytes.Buffer do not implement io.Closer
	stdio := transport.NewGeneric(reader, &writer)
	err := stdio.Close()

	// No error should occur as there's nothing to close
	if err != nil {
		t.Fatalf("Close() error = %v, wantErr %v", err, false)
	}
}

func TestStdio_Close_Error(t *testing.T) {
	t.Parallel()
	readerErr := errors.New("reader close error")
	writerErr := errors.New("writer close error")
	readerMock := newMockReadCloser(strings.NewReader(""), readerErr)
	writerMock := newMockWriteCloser(&bytes.Buffer{}, writerErr)

	stdio := transport.NewGeneric(readerMock, writerMock)
	err := stdio.Close()

	if err == nil {
		t.Fatalf("Close() error = nil, want error")
	}
	// Check if the joined error contains the specific errors
	if !errors.Is(err, readerErr) {
		t.Errorf("Close() error %q does not contain reader error %q", err, readerErr)
	}
	if !errors.Is(err, writerErr) {
		t.Errorf("Close() error %q does not contain writer error %q", err, writerErr)
	}
	if readerMock.getCloseCalls() != 1 {
		t.Errorf("Reader Close calls = %d, want 1", readerMock.getCloseCalls())
	}
	if writerMock.getCloseCalls() != 1 {
		t.Errorf("Writer Close calls = %d, want 1", writerMock.getCloseCalls())
	}
}
