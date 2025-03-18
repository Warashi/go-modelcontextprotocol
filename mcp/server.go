package mcp

import (
	"context"
	"io"
	"os"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
)

type ServerOption func(*Server)

type Server struct {
	conn *jsonrpc2.Conn
}

func NewServer(r io.Reader, w io.Writer, opts ...ServerOption) *Server {
	s := new(Server)
	for _, opt := range opts {
		opt(s)
	}

	var initOpts []jsonrpc2.ConnectionInitializationOption
	initOpts = append(initOpts,
		jsonrpc2.WithHandlerFunc("ping", s.Ping),
		jsonrpc2.WithHandlerFunc("initialize", s.Initialize),
	)

	s.conn = jsonrpc2.NewConnection(r, w, initOpts...)

	return s
}

func NewStdioServer(opts ...ServerOption) *Server {
	return NewServer(os.Stdin, os.Stdout, opts...)
}

func (s *Server) Serve(ctx context.Context) error {
	return s.conn.Serve(ctx)
}

func (s *Server) Close() error {
	return s.conn.Close()
}
