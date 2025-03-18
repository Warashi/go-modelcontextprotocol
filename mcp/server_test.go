package mcp

import (
	"context"
	"io"
	"testing"
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
