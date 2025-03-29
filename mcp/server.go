package mcp

import (
	"context"
	crand "crypto/rand"
	"errors"
	"math/rand/v2"
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
	idSampler   *rand.ChaCha8
	connections map[uint64]*jsonrpc2.Conn
}

// NewServer creates a new MCP server.
func NewServer(name, version string, opts ...ServerOption) (*Server, error) {
	var seed [32]byte
	_, err := crand.Read(seed[:])
	if err != nil {
		return nil, err
	}

	s := &Server{
		name:              name,
		version:           version,
		tools:             make(map[string]tool),
		resources:         make([]Resource, 0),         // to return empty list instead of nil
		resourceTemplates: make([]ResourceTemplate, 0), // to return empty list instead of nil
		idSampler:         rand.NewChaCha8(seed),
		connections:       make(map[uint64]*jsonrpc2.Conn),
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
func (s *Server) HandleSession(ctx context.Context, t transport.Session) (id uint64, err error) {
	return s.Serve(ctx, t)
}

// ServeStdio serves the server over stdin and stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	_, err := s.Serve(ctx, transport.NewStdio())
	return err
}

// Serve starts the server.
func (s *Server) Serve(ctx context.Context, t transport.Session) (uint64, error) {
	conn := jsonrpc2.NewConnection(t, s.initOpts...)

	s.mu.Lock()
	id := s.idSampler.Uint64()
	s.connections[id] = conn
	s.mu.Unlock()

	return id, conn.Serve(ctx)
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
