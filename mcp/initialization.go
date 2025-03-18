package mcp

import (
	"context"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
)

type InitializationRequestParams struct{
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities struct {
		Roots *struct{
			ListChanged bool `json:"listChanged,omitempty,omitzero"`
		} `json:"roots,omitempty,omitzero"`
		Sampling *struct{} `json:"sampling,omitempty,omitzero"`
	} `json:"capabilities,omitzero"`
	ClientInfo struct {
		Name string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type InitializationResponseData struct{
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities struct {
		Logging *struct{} `json:"logging,omitempty,omitzero"`
		Prompts *struct{
			ListChanged bool `json:"listChanged,omitempty,omitzero"`
		} `json:"prompts,omitempty,omitzero"`
		Resources *struct{
			Subscribe bool `json:"subscribe,omitempty,omitzero"`
			ListChanged bool `json:"listChanged,omitempty,omitzero"`
		} `json:"resources,omitempty,omitzero"`
		Tools *struct{
			ListChanged bool `json:"listChanged,omitempty,omitzero"`
		} `json:"tools,omitempty,omitzero"`
	} `json:"capabilities"`
	ServerInfo struct {
		Name string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

func (s *Server) Initialize(ctx context.Context, request *Request[InitializationRequestParams]) (*Result[InitializationResponseData], error) {
	// TODO: Implement this method
	return nil, jsonrpc2.NewError(-32601, "Method not found", struct{}{})
}
