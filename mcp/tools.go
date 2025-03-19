package mcp

import (
	"context"

	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

// Tool is a tool definition in the MCP.
type Tool[Input any] struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Description is the description of the tool.
	Description string `json:"description"`
	// InputSchema is the schema of the tool's input.
	InputSchema jsonschema.Object `json:"inputSchema"`
	// Handler is the handler of the tool.
	Handler ToolHandler[Input] `json:"-"`
}

type ToolHandler[Input any] interface {
	Handle(ctx context.Context, input Input) (any, error)
}

type ToolHandlerFunc[Input any] func(ctx context.Context, input Input) (any, error)

func (f ToolHandlerFunc[Input]) Handle(ctx context.Context, input Input) (any, error) {
	return f(ctx, input)
}
