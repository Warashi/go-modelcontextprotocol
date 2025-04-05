package mcp

import (
	"context"
)

const (
	SupportedProtocolVersion = "2024-11-05"
)

// RootsCapabilities is the capabilities for the roots feature.
type RootsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty,omitzero"`
}

// SamplingCapabilities is the capabilities for the sampling feature.
type SamplingCapabilities struct{}

// ClientInfoData is the data for the client info.
type ClientInfoData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializationRequestCapabilities is the capabilities for the initialization request.
type InitializationRequestCapabilities struct {
	Roots    *RootsCapabilities    `json:"roots,omitempty,omitzero"`
	Sampling *SamplingCapabilities `json:"sampling,omitempty,omitzero"`
}

// InitializationRequestParams is the params for the initialization request.
type InitializationRequestParams struct {
	ProtocolVersion string                            `json:"protocolVersion"`
	Capabilities    InitializationRequestCapabilities `json:"capabilities,omitzero"`
	ClientInfo      ClientInfoData                    `json:"clientInfo"`
}

// LoggingCapabilities is the capabilities for the logging feature.
type LoggingCapabilities struct{}

// PromptsCapabilities is the capabilities for the prompts feature.
type PromptsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty,omitzero"`
}

// ResourcesCapabilities is the capabilities for the resources feature.
type ResourcesCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty,omitzero"`
	ListChanged bool `json:"listChanged,omitempty,omitzero"`
}

// ToolsCapabilities is the capabilities for the tools feature.
type ToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty,omitzero"`
}

// ServerInfoData is the data for the server info.
type ServerInfoData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities is the capabilities for the server.
type Capabilities struct {
	Logging   *LoggingCapabilities   `json:"logging,omitempty,omitzero"`
	Prompts   *PromptsCapabilities   `json:"prompts,omitempty,omitzero"`
	Resources *ResourcesCapabilities `json:"resources,omitempty,omitzero"`
	Tools     *ToolsCapabilities     `json:"tools,omitempty,omitzero"`
}

// InitializationResponseData is the data for the initialization response.
type InitializationResponseData struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ServerInfo      ServerInfoData `json:"serverInfo"`
}

// Initialize initializes the server.
func (s *Server) Initialize(ctx context.Context, request *Request[InitializationRequestParams]) (*Result[InitializationResponseData], error) {
	result := &Result[InitializationResponseData]{
		Data: InitializationResponseData{
			ProtocolVersion: SupportedProtocolVersion,
			Capabilities:    Capabilities{},
			ServerInfo: ServerInfoData{
				Name:    s.name,
				Version: s.version,
			},
		},
	}

	if len(s.tools) > 0 {
		// we have tools
		result.Data.Capabilities.Tools = &ToolsCapabilities{}
	}

	if (len(s.resources) > 0 || len(s.resourceTemplates) > 0) && s.resourceReader != nil {
		// we have resources and a resource reader
		result.Data.Capabilities.Resources = &ResourcesCapabilities{}
	}

	return result, nil
}

func (s *Server) Initialized(ctx context.Context, params struct{}) (struct{}, error) {
	return struct{}{}, nil
}
