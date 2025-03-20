package mcp

import (
	"context"
	"io"
	"os"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
)

// ServerOption is a function that configures a Server.
type ServerOption func(*Server)

// WithCustomHandler sets a custom handler for a method.
// You can use this to override the default handlers.
func WithCustomHandler[Params, Result any](method string, handler jsonrpc2.Handler[Params, Result]) ServerOption {
	return func(s *Server) {
		s.initOpts = append(s.initOpts, jsonrpc2.WithHandler(method, handler))
	}
}

// WithCustomHandlerFunc sets a custom handler for a method.
// You can use this to override the default handlers.
func WithCustomHandlerFunc[Params, Result any](method string, handler func(ctx context.Context, params Params) (Result, error)) ServerOption {
	return func(s *Server) {
		s.initOpts = append(s.initOpts, jsonrpc2.WithHandlerFunc(method, handler))
	}
}

// WithTool sets a tool for the server.
func WithTool[Input, Output any](tool Tool[Input, Output]) ServerOption {
	return func(s *Server) {
		s.tools[tool.Name] = tool
	}
}

// WithResource sets a resource for the server.
func WithResource(resource Resource) ServerOption {
	return func(s *Server) {
		s.resources = append(s.resources, resource)
	}
}

// WithResourceTemplate sets a resource template for the server.
func WithResourceTemplate(template ResourceTemplate) ServerOption {
	return func(s *Server) {
		s.resourceTemplates = append(s.resourceTemplates, template)
	}
}

// WithResourceReader sets a resource reader for the server.
func WithResourceReader(reader ResourceReader) ServerOption {
	return func(s *Server) {
		s.resourceReader = reader
	}
}

// Server is a MCP server.
type Server struct {
	name    string
	version string

	initOpts []jsonrpc2.ConnectionInitializationOption

	tools map[string]tool

	resources         []Resource
	resourceTemplates []ResourceTemplate
	resourceReader    ResourceReader

	conn *jsonrpc2.Conn
}

// NewServer creates a new MCP server.
func NewServer(name, version string, r io.Reader, w io.Writer, opts ...ServerOption) *Server {
	s := &Server{
		name:    name,
		version: version,
		tools:   make(map[string]tool),
	}

	for _, opt := range opts {
		opt(s)
	}

	var initOpts []jsonrpc2.ConnectionInitializationOption
	initOpts = append(initOpts,
		jsonrpc2.WithHandlerFunc("ping", s.Ping),
		jsonrpc2.WithHandlerFunc("initialize", s.Initialize),
		jsonrpc2.WithHandlerFunc("tools/list", s.ListTools),
		jsonrpc2.WithHandlerFunc("tools/call", s.CallTool),
		jsonrpc2.WithHandlerFunc("resources/list", s.ListResources),
		jsonrpc2.WithHandlerFunc("resources/read", s.ReadResource),
		jsonrpc2.WithHandlerFunc("resources/templates/list", s.ListResourceTemplates),
	)

	initOpts = append(initOpts, s.initOpts...)

	s.conn = jsonrpc2.NewConnection(r, w, initOpts...)

	return s
}

// NewStdioServer creates a new MCP server that uses the standard input and output.
func NewStdioServer(name, version string, opts ...ServerOption) *Server {
	return NewServer(name, version, os.Stdin, os.Stdout, opts...)
}

// Serve starts the server.
func (s *Server) Serve(ctx context.Context) error {
	return s.conn.Serve(ctx)
}

// Close closes the server.
func (s *Server) Close() error {
	return s.conn.Close()
}
