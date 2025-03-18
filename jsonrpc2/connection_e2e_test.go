package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type testHandler struct{}

func (h *testHandler) HandleRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
	return map[string]any{"response": "success"}, nil
}

func TestConn_Call(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2, WithHandler("testMethod", &testHandler{}))
	go conn1.Serve(t.Context())
	conn2 := NewConnection(r2, w1)
	conn2.Open()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	params := map[string]any{"param1": "value1"}
	result, err := Call[any, any](ctx, conn2, "testMethod", params)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	expected := map[string]any{"response": "success"}
	if !jsonEqual(result, expected) {
		t.Errorf("Call result = %v; want %v", result, expected)
	}
}

func TestConn_MethodNotFound(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2, WithHandler("testMethod", &testHandler{}))
	go conn1.Serve(t.Context())
	conn2 := NewConnection(r2, w1)
	conn2.Open()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	params := map[string]any{"param1": "value1"}
	_, err := Call[any, any](ctx, conn2, "nonExistentMethod", params)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	expectedErrMsg := "method not found"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %v, got %v", expectedErrMsg, err.Error())
	}
}

func jsonEqual(a, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return bytes.Equal(aj, bj)
}

func TestConn_Notification(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	var called atomic.Bool

	conn1 := NewConnection(r1, w2, WithHandler("testMethod", HandlerFunc[any, any](func(ctx context.Context, req any) (any, error) {
		called.Store(true)
		return nil, nil
	})))
	go conn1.Serve(t.Context())
	conn2 := NewConnection(r2, w1)
	conn2.Open()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	params := map[string]any{"param1": "value1"}
	err := Notify[any](ctx, conn2, "testMethod", params)
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	// Notify is asynchronous, so we need to wait for the handler to be called.
	time.Sleep(1 * time.Millisecond)

	if !called.Load() {
		t.Errorf("Handler not called")
	}
}
