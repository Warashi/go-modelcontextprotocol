package mcp

import (
	"context"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"

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

// ToolCallResult is the result of the tool call.
type ToolCallResult struct {
	IsError bool                    `json:"isError"`
	Content []ToolCallResultContent `json:"content"`
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
func (t *Tool[Input]) Handle(ctx context.Context, input json.RawMessage) (*ToolCallResult, error) {
	if err := t.Validate(input); err != nil {
		return &ToolCallResult{
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
		return &ToolCallResult{
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
		return &ToolCallResult{
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
			return &ToolCallResult{
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

	return &ToolCallResult{
		IsError: false,
		Content: contents,
	}, nil
}

// convertToToolCallResultContent converts the result to the ToolCallResultContent.
// if the result is already a ToolCallResultContent, it returns the result as is.
// if the result is a string, it converts the result to the ToolCallResultTextContent.
// if the result implements encoding.TextMarshaler, calls MarshalText and returns the result as the ToolCallResultTextContent.
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
