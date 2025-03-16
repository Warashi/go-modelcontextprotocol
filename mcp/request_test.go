package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestProgressToken_IsNull(t *testing.T) {
	tests := []struct {
		token    ProgressToken
		expected bool
	}{
		{token: ProgressToken{value: nil}, expected: true},
		{token: ProgressToken{value: "test"}, expected: false},
		{token: ProgressToken{value: 123}, expected: false},
	}

	for _, test := range tests {
		if result := test.token.IsNull(); result != test.expected {
			t.Errorf("ProgressToken(%v).IsNull() = %v; want %v", test.token, result, test.expected)
		}
	}
}

func TestProgressToken_String(t *testing.T) {
	tests := []struct {
		token    ProgressToken
		expected string
	}{
		{token: ProgressToken{value: "test"}, expected: "test"},
		{token: ProgressToken{value: 123}, expected: "123"},
		{token: ProgressToken{value: nil}, expected: ""},
	}

	for _, test := range tests {
		if result := test.token.String(); result != test.expected {
			t.Errorf("ProgressToken(%v).String() = %v; want %v", test.token, result, test.expected)
		}
	}
}

func TestProgressToken_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected ProgressToken
	}{
		{input: `"test"`, expected: ProgressToken{value: "test"}},
		{input: `123`, expected: ProgressToken{value: 123}},
		{input: `null`, expected: ProgressToken{value: nil}},
	}

	for _, test := range tests {
		var token ProgressToken
		if err := json.Unmarshal([]byte(test.input), &token); err != nil {
			t.Errorf("UnmarshalJSON(%v) error: %v", test.input, err)
		}
		if token != test.expected {
			t.Errorf("UnmarshalJSON(%v) = %v; want %v", test.input, token, test.expected)
		}
	}
}

func TestProgressToken_MarshalJSON(t *testing.T) {
	tests := []struct {
		token    ProgressToken
		expected string
	}{
		{token: ProgressToken{value: "test"}, expected: `"test"`},
		{token: ProgressToken{value: 123}, expected: `123`},
		{token: ProgressToken{value: nil}, expected: `null`},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.token)
		if err != nil {
			t.Errorf("MarshalJSON(%v) error: %v", test.token, err)
		}
		if string(result) != test.expected {
			t.Errorf("MarshalJSON(%v) = %v; want %v", test.token, string(result), test.expected)
		}
	}
}

func TestRequest_MarshalJSON(t *testing.T) {
	tests := []struct {
		req      *Request[any]
		expected string
	}{
		{
			req:      &Request[any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: "1"}}, Params: map[string]any{"param1": "value1"}},
			expected: `{"param1":"value1","_meta":{"progressToken":"1"}}`,
		},
		{
			req:      &Request[any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: 1}}, Params: map[string]any{"param1": "value1"}},
			expected: `{"param1":"value1","_meta":{"progressToken":1}}`,
		},
		{
			req:      &Request[any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: nil}}, Params: map[string]any{"param1": "value1"}},
			expected: `{"param1":"value1"}`,
		},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.req)
		if err != nil {
			t.Errorf("MarshalJSON(%v) error: %v", test.req, err)
		}
		assertJSONEqual(t, test.expected, string(result))
	}
}

func TestRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected Request[map[string]any]
	}{
		{
			input:    `{"param1":"value1","_meta":{"progressToken":"1"}}`,
			expected: Request[map[string]any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: "1"}}, Params: map[string]any{"param1": "value1"}},
		},
		{
			input:    `{"param1":"value1","_meta":{"progressToken":1}}`,
			expected: Request[map[string]any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: 1}}, Params: map[string]any{"param1": "value1"}},
		},
		{
			input:    `{"param1":"value1"}`,
			expected: Request[map[string]any]{Meta: RequestMeta{ProgressToken: ProgressToken{value: nil}}, Params: map[string]any{"param1": "value1"}},
		},
	}

	for _, test := range tests {
		var req Request[map[string]any]
		if err := json.Unmarshal([]byte(test.input), &req); err != nil {
			t.Errorf("UnmarshalJSON(%v) error: %v", test.input, err)
		}
		if req.Meta.ProgressToken != test.expected.Meta.ProgressToken || fmt.Sprintf("%v", req.Params) != fmt.Sprintf("%v", test.expected.Params) {
			t.Errorf("UnmarshalJSON(%v) = %v; want %v", test.input, req, test.expected)
		}
	}
}
