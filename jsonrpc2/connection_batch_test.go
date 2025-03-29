package jsonrpc2

import (
	"context"
	"encoding/json"
	"iter"
	"sync"
	"testing"
	"time"
)

// dummyTransport implements transport.Session for testing purposes.
// It records messages sent via Send.

type dummyTransport struct {
	mu           sync.Mutex
	sentMessages [][]byte
}

func (d *dummyTransport) Send(b json.RawMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sentMessages = append(d.sentMessages, b)
	return nil
}

// For these tests, Receive is not used.
func (d *dummyTransport) Receive() iter.Seq[json.RawMessage] {
	return func(yield func(json.RawMessage) bool) {
		for _, msg := range d.sentMessages {
			if !yield(msg) {
				break
			}
		}
	}
}

func (d *dummyTransport) Close() error {
	return nil
}

// lastSentMessage returns the most recent sent message.
func (d *dummyTransport) lastSentMessage() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.sentMessages) == 0 {
		return nil
	}
	return d.sentMessages[len(d.sentMessages)-1]
}

// Test_HandleBatchMessage_SingleRequest tests a batch with a single valid request.
func Test_HandleBatchMessage_SingleRequest(t *testing.T) {
	dt := &dummyTransport{}
	conn := NewConnection(dt)
	// Register handler for "batchTest" that returns "ok".
	RegisterHandler(conn, "batchTest", HandlerFunc[map[string]any, string](func(ctx context.Context, req map[string]any) (string, error) {
		return "ok", nil
	}))

	// Create a valid request: {"jsonrpc": "2.0", "method": "batchTest", "id": 1, "params": {}}
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  "batchTest",
		"id":      1,
		"params":  map[string]any{},
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	// Create batch array with one request.
	batch := []json.RawMessage{reqBytes}
	batchBytes, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}

	// Call handleMessage with the batch message.
	err = conn.handleMessage(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}

	// Expect a batch response with one element.
	last := dt.lastSentMessage()
	if last == nil {
		t.Fatalf("No response sent")
	}
	var responses []map[string]any
	if err := json.Unmarshal(last, &responses); err != nil {
		t.Fatalf("Unmarshal batch response error: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}
	// Check the response has id 1 and result "ok"
	if responses[0]["id"] != float64(1) { // JSON numbers become float64
		t.Errorf("Expected id 1, got %v", responses[0]["id"])
	}
	if responses[0]["result"] != "ok" {
		t.Errorf("Expected result 'ok', got %v", responses[0]["result"])
	}
}

// Test_HandleBatchMessage_NotificationOnly tests a batch containing only a notification; no response expected.
func Test_HandleBatchMessage_NotificationOnly(t *testing.T) {
	dt := &dummyTransport{}
	conn := NewConnection(dt)
	var called bool
	RegisterHandler(conn, "notifyTest", HandlerFunc[map[string]any, any](func(ctx context.Context, req map[string]any) (any, error) {
		called = true
		return nil, nil
	}))

	// Create a notification: {"jsonrpc": "2.0", "method": "notifyTest", "params": {}}
	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifyTest",
		"params":  map[string]any{},
	}
	notifBytes, err := json.Marshal(notif)
	if err != nil {
		t.Fatal(err)
	}
	batch := []json.RawMessage{notifBytes}
	batchBytes, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.handleMessage(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	// Wait a bit to ensure asynchronous processing (if any).
	time.Sleep(10 * time.Millisecond)
	if !called {
		t.Errorf("Notification handler was not called")
	}
	// Since it's notification only, no response should be sent.
	if dt.lastSentMessage() != nil {
		t.Errorf("Expected no response for notification-only batch, but got a response")
	}
}

// Test_HandleBatchMessage_Mixed tests a batch with one request and one notification.
func Test_HandleBatchMessage_Mixed(t *testing.T) {
	dt := &dummyTransport{}
	conn := NewConnection(dt)
	RegisterHandler(conn, "batchTest", HandlerFunc[map[string]any, string](func(ctx context.Context, req map[string]any) (string, error) {
		return "ok", nil
	}))
	var called bool
	RegisterHandler(conn, "notifyTest", HandlerFunc[map[string]any, any](func(ctx context.Context, req map[string]any) (any, error) {
		called = true
		return nil, nil
	}))

	// Create a valid request and a notification.
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  "batchTest",
		"id":      2,
		"params":  map[string]any{},
	}
	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifyTest",
		"params":  map[string]any{},
	}
	reqBytes, _ := json.Marshal(req)
	notifBytes, _ := json.Marshal(notif)
	batch := []json.RawMessage{reqBytes, notifBytes}
	batchBytes, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.handleMessage(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	// Wait a little for the notification to be processed.
	time.Sleep(10 * time.Millisecond)
	if !called {
		t.Errorf("Notification handler was not called")
	}
	// Expect a batch response with one element (for the request).
	last := dt.lastSentMessage()
	if last == nil {
		t.Fatalf("No response sent")
	}
	var responses []map[string]any
	if err := json.Unmarshal(last, &responses); err != nil {
		t.Fatalf("Unmarshal batch response error: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}
	if responses[0]["id"] != float64(2) {
		t.Errorf("Expected id 2, got %v", responses[0]["id"])
	}
	if responses[0]["result"] != "ok" {
		t.Errorf("Expected result 'ok', got %v", responses[0]["result"])
	}
}

// Test_HandleBatchMessage_Empty tests a batch with an empty array which should trigger an error response.
func Test_HandleBatchMessage_Empty(t *testing.T) {
	dt := &dummyTransport{}
	conn := NewConnection(dt)
	// Empty batch: []
	batchBytes := []byte("[]")

	err := conn.handleMessage(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	// Expect an error response (object, not an array) due to empty batch.
	last := dt.lastSentMessage()
	if last == nil {
		t.Fatalf("No response sent for empty batch")
	}
	var resp map[string]any
	if err := json.Unmarshal(last, &resp); err != nil {
		t.Fatalf("Unmarshal response error: %v", err)
	}
	if resp["error"] == nil {
		t.Errorf("Expected error response for empty batch")
	}
	if code, ok := resp["error"].(map[string]any)["code"]; !ok || code != float64(CodeInvalidRequest) {
		t.Errorf("Expected error code %d, got %v", CodeInvalidRequest, code)
	}
}

// Test_HandleBatchMessage_InvalidJSON tests a batch with invalid JSON which should trigger a parse error response.
func Test_HandleBatchMessage_InvalidJSON(t *testing.T) {
	dt := &dummyTransport{}
	conn := NewConnection(dt)
	// Invalid JSON, e.g. "[{"
	invalidJSON := []byte("[{")

	err := conn.handleMessage(context.Background(), invalidJSON)
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	last := dt.lastSentMessage()
	if last == nil {
		t.Fatalf("No response sent for invalid JSON")
	}
	var resp map[string]any
	if err := json.Unmarshal(last, &resp); err != nil {
		t.Fatalf("Unmarshal response error: %v", err)
	}
	if resp["error"] == nil {
		t.Errorf("Expected error response for invalid JSON")
	}
	if code, ok := resp["error"].(map[string]any)["code"]; !ok || code != float64(CodeParseError) {
		t.Errorf("Expected error code %d, got %v", CodeParseError, code)
	}
}
