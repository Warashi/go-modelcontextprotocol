package mcp

import (
	"context"
	"io"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func TestNewServer(t *testing.T) {
	r, w := io.Pipe()
	server := NewServer("test", "1.0.0", r, w)

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}
	if server.conn == nil {
		t.Fatal("expected server.conn to be non-nil")
	}
}

func TestNewStdioServer(t *testing.T) {
	server := NewStdioServer("test", "1.0.0")

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}
	if server.conn == nil {
		t.Fatal("expected server.conn to be non-nil")
	}
}

func TestServer_Serve(t *testing.T) {
	r, w := io.Pipe()
	server := NewServer("test", "1.0.0", r, w)
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
	server := NewServer("test", "1.0.0", r, w)

	if err := server.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestServerOptions(t *testing.T) {
	t.Run("WithCustomHandler", func(t *testing.T) {
		r, w := io.Pipe()
		handler := jsonrpc2.HandlerFunc[string, string](func(ctx context.Context, params string) (string, error) {
			return params + "_response", nil
		})
		server := NewServer("test", "1.0.0", r, w, WithCustomHandler("test_method", handler))

		if len(server.initOpts) != 1 { // Only the custom handler
			t.Errorf("expected 1 handler, got %d", len(server.initOpts))
		}
	})

	t.Run("WithCustomHandlerFunc", func(t *testing.T) {
		r, w := io.Pipe()
		handlerFunc := func(ctx context.Context, params string) (string, error) {
			return params + "_response", nil
		}
		server := NewServer("test", "1.0.0", r, w, WithCustomHandlerFunc("test_method", handlerFunc))

		if len(server.initOpts) != 1 { // Only the custom handler
			t.Errorf("expected 1 handler, got %d", len(server.initOpts))
		}
	})

	t.Run("WithTool", func(t *testing.T) {
		r, w := io.Pipe()
		tool := NewToolFunc(
			"test_tool",
			"Test tool description",
			jsonschema.Object{},
			func(ctx context.Context, input string) (string, error) {
				return input + "_processed", nil
			},
		)
		server := NewServer("test", "1.0.0", r, w, WithTool(tool))

		if len(server.tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(server.tools))
		}
		if _, ok := server.tools["test_tool"]; !ok {
			t.Error("expected test_tool to be registered")
		}
	})

	t.Run("WithResource", func(t *testing.T) {
		r, w := io.Pipe()
		resource := Resource{
			URI:  "test://resource",
			Name: "test_resource",
		}
		server := NewServer("test", "1.0.0", r, w, WithResource(resource))

		if len(server.resources) != 1 {
			t.Errorf("expected 1 resource, got %d", len(server.resources))
		}
		if server.resources[0].Name != "test_resource" {
			t.Errorf("expected resource name to be test_resource, got %s", server.resources[0].Name)
		}
	})

	t.Run("WithResourceTemplate", func(t *testing.T) {
		r, w := io.Pipe()
		template := ResourceTemplate{
			URITemplate: "test://template/{name}",
			Name:        "test_template",
		}
		server := NewServer("test", "1.0.0", r, w, WithResourceTemplate(template))

		if len(server.resourceTemplates) != 1 {
			t.Errorf("expected 1 resource template, got %d", len(server.resourceTemplates))
		}
		if server.resourceTemplates[0].Name != "test_template" {
			t.Errorf("expected template name to be test_template, got %s", server.resourceTemplates[0].Name)
		}
	})

	t.Run("WithResourceReader", func(t *testing.T) {
		r, w := io.Pipe()
		reader := NewResourceReaderMux()
		server := NewServer("test", "1.0.0", r, w, WithResourceReader(reader))

		if server.resourceReader == nil {
			t.Error("expected resource reader to be set")
		}
	})
}
