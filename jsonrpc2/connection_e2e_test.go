package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestConn_WithHandlerFunc(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	called := false
	conn1 := NewConnection(r1, w2, WithHandlerFunc("testMethod", func(ctx context.Context, req any) (any, error) {
		called = true
		return map[string]any{"response": "success"}, nil
	}))
	go conn1.Serve(t.Context())
	conn2 := NewConnection(r2, w1)
	conn2.Open()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	params := map[string]any{"param1": "value1"}
	result, err := Call[any, any, any](ctx, conn2, "testMethod", params)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	expected := map[string]any{"response": "success"}
	if !jsonEqual(result, expected) {
		t.Errorf("Call result = %v; want %v", result, expected)
	}

	if !called {
		t.Error("HandlerFunc was not called")
	}
}

func TestConn_Close(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2)
	conn2 := NewConnection(r2, w1)

	// Test closing multiple times
	if err := conn1.Close(); err != nil {
		t.Errorf("First Close failed: %v", err)
	}

	if err := conn1.Close(); err != nil {
		t.Errorf("Second Close should return nil, got: %v", err)
	}

	// Test that operations after close fail
	if err := conn1.Open(); err == nil {
		t.Error("Open after Close should fail")
	}

	if err := conn1.Serve(context.Background()); err == nil {
		t.Error("Serve after Close should fail")
	}

	// Clean up
	conn2.Close()
}

func TestConn_ServeErrors(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2)
	conn2 := NewConnection(r2, w1)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := conn1.Serve(ctx); err == nil {
		t.Error("Serve with cancelled context should fail")
	}

	// Test connection closed
	conn1.Close()
	if err := conn1.Serve(context.Background()); err == nil {
		t.Error("Serve with closed connection should fail")
	}

	// Clean up
	conn2.Close()
}

func TestConn_CallErrors(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2)
	conn2 := NewConnection(r2, w1)
	go conn1.Serve(context.Background())
	conn2.Open()

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Call[any, any](ctx, conn2, "testMethod", struct{}{})
	if err == nil {
		t.Error("Call with cancelled context should fail")
	}

	// Test invalid method
	ctx = context.Background()
	_, err = Call[any, any](ctx, conn2, "nonExistentMethod", struct{}{})
	if err == nil {
		t.Error("Call with invalid method should fail")
	}

	// Test connection closed
	conn2.Close()
	_, err = Call[any, any](ctx, conn2, "testMethod", struct{}{})
	if err == nil {
		t.Error("Call with closed connection should fail")
	}

	// Clean up
	conn1.Close()
}

func TestConn_SendResponseAndError(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := NewConnection(r1, w2)
	conn2 := NewConnection(r2, w1)
	go conn1.Serve(context.Background())

	// Test sendResponse with null ID
	ctx := context.Background()
	err := conn1.sendResponse(ctx, ID{value: nil}, nil)
	if err == nil {
		t.Error("sendResponse with null ID should fail")
	}

	// Test sendError with null ID
	err = conn1.sendError(ctx, ID{value: nil}, errors.New("test error"))
	if err == nil {
		t.Error("sendError with null ID should fail")
	}

	// Test sendResponse with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = conn1.sendResponse(ctx, NewID("test"), nil)
	if err == nil {
		t.Error("sendResponse with cancelled context should fail")
	}

	// Test sendError with cancelled context
	err = conn1.sendError(ctx, NewID("test"), errors.New("test error"))
	if err == nil {
		t.Error("sendError with cancelled context should fail")
	}

	// Clean up
	conn1.Close()
	conn2.Close()
}
