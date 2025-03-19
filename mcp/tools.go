package mcp

import (
	"context"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
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
	NextCursor string `json:"nextCursor"`
}

// ListTools implements the jsonrpc2.HandlerFunc
func (s *Server) ListTools(ctx context.Context, request *Request[ListToolsRequestParams]) (*Result[ListToolsResultData], error) {
	if request.Params.Cursor != "" {
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
	IsError bool                    `json:"isError"`
	Content []ToolCallResultContent `json:"content"`
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

// ToolHandler is the handler of the tool.
type ToolHandler[Input any] interface {
	Handle(ctx context.Context, input Input) ([]any, error)
}

// ToolHandlerFunc is a function that implements ToolHandler.
type ToolHandlerFunc[Input any] func(ctx context.Context, input Input) ([]any, error)

// Handle implements ToolHandler.
func (f ToolHandlerFunc[Input]) Handle(ctx context.Context, input Input) ([]any, error) {
	return f(ctx, input)
}

type ToolCallResultContent interface {
	isToolCallResultContent()
}

// ToolCallResultTextContent is the text content of the tool call result.
type ToolCallResultTextContent struct {
	Text string `json:"text"`
}

// isToolCallResultContent implements ToolCallResultContent.
func (ToolCallResultTextContent) isToolCallResultContent() {}

// MarshalJSON implements json.Marshaler.
func (t *ToolCallResultTextContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": "text",
		"text": t.Text,
	})
}

// ToolCallResultImageContent is the image content of the tool call result.
type ToolCallResultImageContent struct {
	Data     []byte `json:"data"`
	MimeType string `json:"mimeType"`
}

// isToolCallResultContent implements ToolCallResultContent.
func (ToolCallResultImageContent) isToolCallResultContent() {}

// MarshalJSON implements json.Marshaler.
func (t *ToolCallResultImageContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":     "image",
		"data":     base64.StdEncoding.EncodeToString(t.Data),
		"mimeType": t.MimeType,
	})
}

// ToolCallResultEmbeddedResourceContent is the embedded resource content of the tool call result.
type ToolCallResultEmbeddedResourceContent struct {
	// TODO: 埋め込みリソースの型を定義する
}

// isToolCallResultContent implements ToolCallResultContent.
func (ToolCallResultEmbeddedResourceContent) isToolCallResultContent() {}

// MarshalJSON implements json.Marshaler.
func (t *ToolCallResultEmbeddedResourceContent) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Handle handles the tool call.
func (t *Tool[Input]) Handle(ctx context.Context, input json.RawMessage) (*ToolCallResultData, error) {
	if err := t.Validate(input); err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []ToolCallResultContent{
				&ToolCallResultTextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	var inputInput Input
	if err := json.Unmarshal(input, &inputInput); err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []ToolCallResultContent{
				&ToolCallResultTextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	result, err := t.Handler.Handle(ctx, inputInput)
	if err != nil {
		return &ToolCallResultData{
			IsError: true,
			Content: []ToolCallResultContent{
				&ToolCallResultTextContent{
					Text: err.Error(),
				},
			},
		}, nil
	}

	contents := make([]ToolCallResultContent, 0, len(result))
	for _, r := range result {
		content, err := convertToToolCallResultContent(r)
		if err != nil {
			return &ToolCallResultData{
				IsError: true,
				Content: []ToolCallResultContent{
					&ToolCallResultTextContent{
						Text: err.Error(),
					},
				},
			}, nil
		}
		contents = append(contents, content)
	}

	return &ToolCallResultData{
		IsError: false,
		Content: contents,
	}, nil
}

// convertToToolCallResultContent converts the result to the ToolCallResultContent.
// if the result is already a ToolCallResultContent, it returns the result as is.
// if the result is a string, it converts the result to the ToolCallResultTextContent.
// if the result implements encoding.TextMarshaler, calls MarshalText and returns the result as the ToolCallResultTextContent.
// if the result implements fmt.Stringer, it returns the result as the ToolCallResultTextContent.
// otherwise, it calls json.Marshal and returns the result as the ToolCallResultTextContent.
func convertToToolCallResultContent(v any) (ToolCallResultContent, error) {
	switch v := v.(type) {
	case string:
		return &ToolCallResultTextContent{
			Text: v,
		}, nil
	case *ToolCallResultTextContent:
		return v, nil
	case *ToolCallResultImageContent:
		return v, nil
	case *ToolCallResultEmbeddedResourceContent:
		return v, nil
	case encoding.TextMarshaler:
		text, err := v.MarshalText()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal text: %w", err)
		}
		return &ToolCallResultTextContent{
			Text: string(text),
		}, nil
	case fmt.Stringer:
		return &ToolCallResultTextContent{
			Text: v.String(),
		}, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		return &ToolCallResultTextContent{
			Text: string(b),
		}, nil
	}
}

// Validate validates the input.
func (t *Tool[Input]) Validate(v json.RawMessage) error {
	var input any
	if err := json.Unmarshal(v, &input); err != nil {
		return err
	}

	return t.InputSchema.Validate(input)
}
