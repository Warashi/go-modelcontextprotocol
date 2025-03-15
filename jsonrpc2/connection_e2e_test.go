package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

	conn1 := NewConnection(r1, w2)
	conn2 := NewConnection(r2, w1)

	RegisterHandler(conn1, "testMethod", &testHandler{})

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

func jsonEqual(a, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return bytes.Equal(aj, bj)
}
