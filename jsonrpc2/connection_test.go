package jsonrpc2

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestID_IsNull(t *testing.T) {
	tests := []struct {
		id       ID
		expected bool
	}{
		{id: ID{value: nil}, expected: true},
		{id: ID{value: "test"}, expected: false},
		{id: ID{value: 123}, expected: false},
	}

	for _, test := range tests {
		if result := test.id.IsNull(); result != test.expected {
			t.Errorf("ID(%v).IsNull() = %v; want %v", test.id, result, test.expected)
		}
	}
}

func TestID_String(t *testing.T) {
	tests := []struct {
		id       ID
		expected string
	}{
		{id: ID{value: "test"}, expected: "test"},
		{id: ID{value: 123}, expected: "123"},
		{id: ID{value: nil}, expected: ""},
	}

	for _, test := range tests {
		if result := test.id.String(); result != test.expected {
			t.Errorf("ID(%v).String() = %v; want %v", test.id, result, test.expected)
		}
	}
}

func TestID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected ID
	}{
		{input: `"test"`, expected: ID{value: "test"}},
		{input: `123`, expected: ID{value: 123}},
		{input: `null`, expected: ID{value: nil}},
	}

	for _, test := range tests {
		var id ID
		if err := json.Unmarshal([]byte(test.input), &id); err != nil {
			t.Errorf("UnmarshalJSON(%v) error: %v", test.input, err)
		}
		if id != test.expected {
			t.Errorf("UnmarshalJSON(%v) = %v; want %v", test.input, id, test.expected)
		}
	}
}

func TestID_MarshalJSON(t *testing.T) {
	tests := []struct {
		id       ID
		expected string
	}{
		{id: ID{value: "test"}, expected: `"test"`},
		{id: ID{value: 123}, expected: `123`},
		{id: ID{value: nil}, expected: `null`},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.id)
		if err != nil {
			t.Errorf("MarshalJSON(%v) error: %v", test.id, err)
		}
		if string(result) != test.expected {
			t.Errorf("MarshalJSON(%v) = %v; want %v", test.id, string(result), test.expected)
		}
	}
}

func TestRequest_MarshalJSON(t *testing.T) {
	tests := []struct {
		req      *Request[any]
		expected string
	}{
		{
			req:      &Request[any]{ID: ID{value: "1"}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
			expected: `{"jsonrpc":"2.0","id":"1","method":"testMethod","params":{"param1":"value1"}}`,
		},
		{
			req:      &Request[any]{ID: ID{value: 1}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
			expected: `{"jsonrpc":"2.0","id":1,"method":"testMethod","params":{"param1":"value1"}}`,
		},
		{
			req:      &Request[any]{ID: ID{value: nil}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
			expected: `{"jsonrpc":"2.0","method":"testMethod","params":{"param1":"value1"}}`,
		},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.req)
		if err != nil {
			t.Errorf("MarshalJSON(%v) error: %v", test.req, err)
		}
		if string(result) != test.expected {
			t.Errorf("MarshalJSON(%v) = %v; want %v", test.req, string(result), test.expected)
		}
	}
}

func TestRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected Request[map[string]any]
	}{
		{
			input:    `{"jsonrpc":"2.0","id":"1","method":"testMethod","params":{"param1":"value1"}}`,
			expected: Request[map[string]any]{ID: ID{value: "1"}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
		},
		{
			input:    `{"jsonrpc":"2.0","id":1,"method":"testMethod","params":{"param1":"value1"}}`,
			expected: Request[map[string]any]{ID: ID{value: 1}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
		},
		{
			input:    `{"jsonrpc":"2.0","method":"testMethod","params":{"param1":"value1"}}`,
			expected: Request[map[string]any]{ID: ID{value: nil}, Method: "testMethod", Params: map[string]any{"param1": "value1"}},
		},
	}

	for _, test := range tests {
		var req Request[map[string]any]
		if err := json.Unmarshal([]byte(test.input), &req); err != nil {
			t.Errorf("UnmarshalJSON(%v) error: %v", test.input, err)
		}
		if req.ID != test.expected.ID || req.Method != test.expected.Method || fmt.Sprintf("%v", req.Params) != fmt.Sprintf("%v", test.expected.Params) {
			t.Errorf("UnmarshalJSON(%v) = %v; want %v", test.input, req, test.expected)
		}
	}
}
