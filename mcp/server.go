package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
	"github.com/Warashi/go-modelcontextprotocol/transport"
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

// WithLogger sets a logger for the server.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
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

	mu          sync.Mutex
	connections map[uint64]*jsonrpc2.Conn
	logger      *slog.Logger
}

// NewServer creates a new MCP server.
func NewServer(name, version string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		name:              name,
		version:           version,
		tools:             make(map[string]tool),
		resources:         make([]Resource, 0),         // to return empty list instead of nil
		resourceTemplates: make([]ResourceTemplate, 0), // to return empty list instead of nil
		connections:       make(map[uint64]*jsonrpc2.Conn),
		logger:            slog.New(slog.DiscardHandler),
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
		jsonrpc2.WithLogger(s.logger),
	)

	// append custom init opts after default handlers
	initOpts = append(initOpts, s.initOpts...)

	// set init opts
	s.initOpts = initOpts

	return s, nil
}

// SSEHandler returns a handler for the SSE transport.
func (s *Server) SSEHandler(baseURL string) (http.Handler, error) {
	return transport.NewSSE(baseURL, s)
}

// HandleSession handles a session.
func (s *Server) HandleSession(ctx context.Context, id uint64, t transport.Session) error {
	return s.Serve(ctx, id, t)
}

// ServeStdio serves the server over stdin and stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Serve(ctx, 0, transport.NewStdio())
}

// Serve starts the server.
func (s *Server) Serve(ctx context.Context, id uint64, t transport.Session) error {
	conn := jsonrpc2.NewConnection(t, s.initOpts...)

	s.mu.Lock()
	s.connections[id] = conn
	s.mu.Unlock()

	return conn.Serve(ctx)
}

// Close closes the server.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	for _, conn := range s.connections {
		err = errors.Join(err, conn.Close())
	}

	return err
}
