package mcp

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
)

func TestServer_Ping(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	server := NewServer("test", "1.0.0", r1, w2)
	go server.Serve(context.Background())

	client := jsonrpc2.NewConnection(r2, w1)
	client.Open()

	result, err := jsonrpc2.Call[any, any](t.Context(), client, "ping", struct{}{})
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if !reflect.DeepEqual(result, map[string]any{}) {
		t.Errorf("Call() result = %v, want %v", result, map[string]any{})
	}
}
