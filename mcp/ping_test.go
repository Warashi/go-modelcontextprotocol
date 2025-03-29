package mcp

import (
	"context"
	"reflect"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
	"github.com/Warashi/go-modelcontextprotocol/transport"
)

func TestServer_Ping(t *testing.T) {
	a, b := transport.NewPipe()

	server := mustNewServer(t, "test", "1.0.0")
	go server.Serve(context.Background(), 1, a)

	client := jsonrpc2.NewConnection(b)
	client.Open()

	result, err := jsonrpc2.Call[any, any](t.Context(), client, "ping", struct{}{})
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if !reflect.DeepEqual(result, map[string]any{}) {
		t.Errorf("Call() result = %v, want %v", result, map[string]any{})
	}
}
