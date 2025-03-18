package mcp

import "context"

func (s *Server) Ping(ctx context.Context, _ struct{}) (struct{}, error) {
	return struct{}{}, nil
}
