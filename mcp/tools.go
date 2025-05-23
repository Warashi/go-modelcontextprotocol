package mcp

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

// tool is utility type to define tool without type parameters.
type tool interface {
	Handle(ctx context.Context, input json.RawMessage) (*ToolCallResultData, error)
}

// ListToolsRequestParams is the parameters of the list tools request.
type ListToolsRequestParams struct {
	Cursor string `json:"cursor"`
}

// ListToolsResultData is the result of the list tools request.
type ListToolsResultData struct {
	Tools      []tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ListTools implements the jsonrpc2.HandlerFunc
func (s *Server) ListTools(ctx context.Context, request *Request[ListToolsRequestParams]) (*Result[ListToolsResultData], error) {
	if request == nil {
		request = &Request[ListToolsRequestParams]{}
	}

	// Initialize empty params if not provided
	var params ListToolsRequestParams
	if request.Params != (ListToolsRequestParams{}) {
		params = request.Params
	}

	if params.Cursor != "" {
		return nil, jsonrpc2.NewError(jsonrpc2.CodeInvalidRequest, "cursor is not supported", struct{}{})
	}

	tools := make([]tool, 0, len(s.tools))
	for _, t := range slices.Sorted(maps.Keys(s.tools)) {
		tools = append(tools, s.tools[t])
	}

	return &Result[ListToolsResultData]{
		Data: ListToolsResultData{
			Tools:      tools,
			NextCursor: "",
		},
	}, nil
}

// ToolCallRequestParams is the parameters of the tool call request.
type ToolCallRequestParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolCallResultData is the result of the tool call.
type ToolCallResultData struct {
	IsError bool        `json:"isError"`
	Content []IsContent `json:"content"`
}

// CallTool implements the jsonrpc2.HandlerFunc
func (s *Server) CallTool(ctx context.Context, request *Request[ToolCallRequestParams]) (*Result[ToolCallResultData], error) {
	tool, ok := s.tools[request.Params.Name]
	if !ok {
		return nil, jsonrpc2.NewError(jsonrpc2.CodeMethodNotFound, "tool not found", struct{}{})
	}

	result, err := tool.Handle(ctx, request.Params.Arguments)
	if err != nil {
		return nil, err
	}

	return &Result[ToolCallResultData]{
		Data: *result,
	}, nil
}

// Tool is a tool definition in the MCP.
type Tool[Input, Output any] struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Description is the description of the tool.
	Description string `json:"description"`
	// InputSchema is the schema of the tool's input.
	InputSchema jsonschema.Object `json:"inputSchema"`
	// Handler is the handler of the tool.
	Handler ToolHandler[Input, Output] `json:"-"`
}

// NewTool creates a new tool.
func NewTool[Input, Output any](name, description string, inputSchema jsonschema.Object, handler ToolHandler[Input, Output]) Tool[Input, Output] {
	return Tool[Input, Output]{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		Handler:     handler,
	}
}

// NewToolFunc creates a new tool with a handler function.
func NewToolFunc[Input, Output any](name, description string, inputSchema jsonschema.Object, handler func(ctx context.Context, input Input) (Output, error)) Tool[Input, Output] {
	// Initialize empty Properties map if not set
	if inputSchema.Properties == nil {
		inputSchema.Properties = make(map[string]jsonschema.Schema)
	}
	return NewTool(name, description, inputSchema, ToolHandlerFunc[Input, Output](handler))
}

// ToolHandler is the handler of the tool.
// If Handle *ToolCallResultData, it returns the result as is.
// If Handle returns a slice, it converts each element to the Content type.
// Otherwise, it returns the result as the ToolCallResultData with single Content.
type ToolHandler[Input, Output any] interface {
	Handle(ctx context.Context, input Input) (Output, error)
}

// ToolHandlerFunc is a function that implements ToolHandler.
type ToolHandlerFunc[Input, Output any] func(ctx context.Context, input Input) (Output, error)

// Handle implements ToolHandler.
func (f ToolHandlerFunc[Input, Output]) Handle(ctx context.Context, input Input) (Output, error) {
	return f(ctx, input)
}

// Validate validates the input.
func (t Tool[Input, Output]) Validate(v json.RawMessage) error {
	return t.InputSchema.Validate(v)
}

// Handle handles the tool call.
func (t Tool[Input, Output]) Handle(ctx context.Context, input json.RawMessage) (*ToolCallResultData, error) {
	if err := t.Validate(input); err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []IsContent{
				&TextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	var inputInput Input
	if err := json.Unmarshal(input, &inputInput); err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []IsContent{
				&TextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	result, err := t.Handler.Handle(ctx, inputInput)
	if err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []IsContent{
				&TextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	return convert(result), nil
}

// convert converts the result to the ToolCallResultData.
// if the result is already a ToolCallResultData, it returns the result as is.
// if the result is a slice, it converts each element.
// otherwise, it returns the result as the ToolCallResultData with single content.
func convert(v any) *ToolCallResultData {
	// Check if the result is already a ToolCallResultData
	if result, ok := v.(*ToolCallResultData); ok {
		return result
	}

	// Use reflection to handle different types of results
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		contents := make([]IsContent, rv.Len())
		for i := range rv.Len() {
			content, err := convertToContent(rv.Index(i).Interface())
			if err != nil {
				return &ToolCallResultData{
					IsError: true,
					Content: []IsContent{
						&TextContent{
							Text: err.Error(),
						},
					},
				}
			}
			contents[i] = content
		}
		return &ToolCallResultData{
			IsError: false,
			Content: contents,
		}
	default:
		content, err := convertToContent(v)
		if err != nil {
			return &ToolCallResultData{
				IsError: true,
				Content: []IsContent{
					&TextContent{
						Text: err.Error(),
					},
				},
			}
		}
		return &ToolCallResultData{
			IsError: false,
			Content: []IsContent{content},
		}
	}
}

// convertToContent converts the result to the ToolCallResultContent.
// if the result is already a ToolCallResultContent, it returns the result as is.
// if the result is a string, it converts the result to the ToolCallResultTextContent.
// if the result implements encoding.TextMarshaler, calls MarshalText and returns the result as the ToolCallResultTextContent.
// if the result implements fmt.Stringer, it returns the result as the ToolCallResultTextContent.
// otherwise, it calls json.Marshal and returns the result as the ToolCallResultTextContent.
func convertToContent(v any) (IsContent, error) {
	switch v := v.(type) {
	case string:
		return &TextContent{
			Text: v,
		}, nil
	case IsContent:
		return v, nil
	case encoding.TextMarshaler:
		text, err := v.MarshalText()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal text: %w", err)
		}
		return &TextContent{
			Text: string(text),
		}, nil
	case fmt.Stringer:
		return &TextContent{
			Text: v.String(),
		}, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		return &TextContent{
			Text: string(b),
		}, nil
	}
}
