package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonrpc2"
	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func TestServer_ListTools(t *testing.T) {
	ctx := context.Background()

	// Test case 1: Empty tools
	server := mustNewServer(t, "test", "1.0.0")

	result, err := server.ListTools(ctx, &Request[ListToolsRequestParams]{
		Params: ListToolsRequestParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"tools":[]}`, string(got))

	// Test case 2: With tools
	tool1 := NewToolFunc("tool1", "Test tool 1", jsonschema.Object{}, func(ctx context.Context, input string) (string, error) {
		return "result1", nil
	})
	tool2 := NewToolFunc("tool2", "Test tool 2", jsonschema.Object{}, func(ctx context.Context, input string) (string, error) {
		return "result2", nil
	})

	server = mustNewServer(t, "test", "1.0.0",
		WithTool(tool1),
		WithTool(tool2),
	)

	result, err = server.ListTools(ctx, &Request[ListToolsRequestParams]{
		Params: ListToolsRequestParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err = json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"tools":[{"name":"tool1","description":"Test tool 1","inputSchema":{"type":"object","additionalProperties":false,"properties":{}}},{"name":"tool2","description":"Test tool 2","inputSchema":{"type":"object","additionalProperties":false,"properties":{}}}]}`, string(got))

	// Test case 3: With cursor (should return error)
	_, err = server.ListTools(ctx, &Request[ListToolsRequestParams]{
		Params: ListToolsRequestParams{
			Cursor: "some-cursor",
		},
	})
	if err == nil {
		t.Error("expected error for non-empty cursor, got nil")
		return
	}
	if jsonrpc2err, ok := err.(jsonrpc2.Error[struct{}]); !ok {
		t.Errorf("unexpected error type: got %T, want jsonrpc2.Error[struct{}]", err)
		return
	} else if jsonrpc2err.Code != jsonrpc2.CodeInvalidRequest {
		t.Errorf("unexpected error code: got %v, want %v", jsonrpc2err.Code, jsonrpc2.CodeInvalidRequest)
	}
}

func TestServer_CallTool(t *testing.T) {
	ctx := context.Background()

	// Test case 1: Tool not found
	server := mustNewServer(t, "test", "1.0.0")

	_, err := server.CallTool(ctx, &Request[ToolCallRequestParams]{
		Params: ToolCallRequestParams{
			Name:      "nonexistent",
			Arguments: json.RawMessage(`{}`),
		},
	})
	if err == nil {
		t.Error("expected error for nonexistent tool, got nil")
		return
	}
	if jsonrpc2err, ok := err.(jsonrpc2.Error[struct{}]); !ok {
		t.Errorf("unexpected error type: got %T, want jsonrpc2.Error[struct{}]", err)
		return
	} else if jsonrpc2err.Code != jsonrpc2.CodeMethodNotFound {
		t.Errorf("unexpected error code: got %v, want %v", jsonrpc2err.Code, jsonrpc2.CodeMethodNotFound)
	}

	// Test case 2: Successful tool call
	schema := jsonschema.Object{
		Properties: map[string]jsonschema.Schema{
			"key": jsonschema.String{},
		},
		Required: []string{"key"},
	}
	tool := NewToolFunc("test", "Test tool", schema, func(ctx context.Context, input map[string]string) (string, error) {
		return "success", nil
	})
	server = mustNewServer(t, "test", "1.0.0",
		WithTool(tool),
	)

	result, err := server.CallTool(ctx, &Request[ToolCallRequestParams]{
		Params: ToolCallRequestParams{
			Name:      "test",
			Arguments: json.RawMessage(`{"key":"value"}`),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"isError":false,"content":[{"type":"text","text":"success"}]}`, string(got))
}

func TestTool_Validate(t *testing.T) {
	schema := jsonschema.Object{
		Properties: map[string]jsonschema.Schema{
			"name": jsonschema.String{},
		},
		Required: []string{"name"},
	}

	tool := NewToolFunc("test", "Test tool", schema, func(ctx context.Context, input map[string]string) (string, error) {
		return input["name"], nil
	})

	// Test case 1: Valid input
	err := tool.Validate(json.RawMessage(`{"name": "test"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test case 2: Invalid input (missing required field)
	err = tool.Validate(json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}

func TestConvert(t *testing.T) {
	// Test case 1: String
	result := convert("test")
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"isError":false,"content":[{"type":"text","text":"test"}]}`, string(got))

	// Test case 2: Slice
	result = convert([]string{"test1", "test2"})
	got, err = json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"isError":false,"content":[{"type":"text","text":"test1"},{"type":"text","text":"test2"}]}`, string(got))

	// Test case 3: Custom IsContent
	content := &TextContent{Text: "custom"}
	result = convert(content)
	got, err = json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	assertJSONEqual(t, `{"isError":false,"content":[{"type":"text","text":"custom"}]}`, string(got))
}
