package mcp

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
)

func TestNewServer(t *testing.T) {
	r, w := io.Pipe()
	server := NewServer(r, w)

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}
	if server.conn == nil {
		t.Fatal("expected server.conn to be non-nil")
	}
}

func TestNewStdioServer(t *testing.T) {
	server := NewStdioServer()

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}
	if server.conn == nil {
		t.Fatal("expected server.conn to be non-nil")
	}
}

func TestServer_Serve(t *testing.T) {
	r, w := io.Pipe()
	server := NewServer(r, w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Serve(ctx); err != nil && err != context.Canceled {
			t.Errorf("Serve() error = %v", err)
		}
	}()

	cancel()
}

func TestServer_Close(t *testing.T) {
	r, w := io.Pipe()
	server := NewServer(r, w)

	if err := server.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestServer_Ping(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	server := NewServer(r1, w2)
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
