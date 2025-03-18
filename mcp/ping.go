package mcp

import "context"

func (s *Server) Ping(ctx context.Context, _ *Request[struct{}]) (*Result[struct{}], error) {
	return &Result[struct{}]{}, nil
}
