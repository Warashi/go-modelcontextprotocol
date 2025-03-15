package jsonrpc2

import (
	"encoding/json"
	"errors"
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

func TestResponse_MarshalJSON(t *testing.T) {
  tests := []struct {
    resp     *Response[any, any]
    expected string
  }{
    {
      resp:     &Response[any, any]{ID: ID{value: "1"}, Result: "result", Error: Error[any]{}},
      expected: `{"jsonrpc":"2.0","id":"1","result":"result"}`,
    },
    {
      resp:     &Response[any, any]{ID: ID{value: 1}, Result: 123, Error: Error[any]{}},
      expected: `{"jsonrpc":"2.0","id":1,"result":123}`,
    },
    {
      resp:     &Response[any, any]{ID: ID{value: nil}, Result: nil, Error: Error[any]{Code: -32000, Message: "error"}},
      expected: `{"jsonrpc":"2.0","error":{"code":-32000,"message":"error"}}`,
    },
  }

  for _, test := range tests {
    result, err := json.Marshal(test.resp)
    if err != nil {
      t.Errorf("MarshalJSON(%v) error: %v", test.resp, err)
    }
    if string(result) != test.expected {
      t.Errorf("MarshalJSON(%v) = %v; want %v", test.resp, string(result), test.expected)
    }
  }
}

func TestResponse_UnmarshalJSON(t *testing.T) {
  tests := []struct {
    input    string
    expected Response[any, any]
  }{
    {
      input:    `{"jsonrpc":"2.0","id":"1","result":"result"}`,
      expected: Response[any, any]{ID: ID{value: "1"}, Result: "result", Error: Error[any]{}},
    },
    {
      input:    `{"jsonrpc":"2.0","id":1,"result":123}`,
      expected: Response[any, any]{ID: ID{value: 1}, Result: 123, Error: Error[any]{}},
    },
    {
      input:    `{"jsonrpc":"2.0","error":{"code":-32000,"message":"error"}}`,
      expected: Response[any, any]{ID: ID{value: nil}, Result: nil, Error: Error[any]{Code: -32000, Message: "error"}},
    },
  }

  for _, test := range tests {
    var resp Response[any, any]
    if err := json.Unmarshal([]byte(test.input), &resp); err != nil {
      t.Errorf("UnmarshalJSON(%v) error: %v", test.input, err)
    }
    if resp.ID != test.expected.ID || fmt.Sprintf("%v", resp.Result) != fmt.Sprintf("%v", test.expected.Result) || resp.Error.Code != test.expected.Error.Code || resp.Error.Message != test.expected.Error.Message {
      t.Errorf("UnmarshalJSON(%v) = %v; want %v", test.input, resp, test.expected)
    }
  }
}

func TestError_Error(t *testing.T) {
  tests := []struct {
    err      Error[any]
    expected string
  }{
    {err: Error[any]{Code: -32000, Message: "error"}, expected: "error"},
    {err: Error[any]{Code: -32001, Message: "another error"}, expected: "another error"},
  }

  for _, test := range tests {
    if result := test.err.Error(); result != test.expected {
      t.Errorf("Error(%v).Error() = %v; want %v", test.err, result, test.expected)
    }
  }
}

func TestError_code(t *testing.T) {
  tests := []struct {
    err      Error[any]
    expected int
  }{
    {err: Error[any]{Code: -32000, Message: "error"}, expected: -32000},
    {err: Error[any]{Code: -32001, Message: "another error"}, expected: -32001},
  }

  for _, test := range tests {
    if result := test.err.code(); result != test.expected {
      t.Errorf("Error(%v).code() = %v; want %v", test.err, result, test.expected)
    }
  }
}

func TestError_message(t *testing.T) {
  tests := []struct {
    err      Error[any]
    expected string
  }{
    {err: Error[any]{Code: -32000, Message: "error"}, expected: "error"},
    {err: Error[any]{Code: -32001, Message: "another error"}, expected: "another error"},
  }

  for _, test := range tests {
    if result := test.err.message(); result != test.expected {
      t.Errorf("Error(%v).message() = %v; want %v", test.err, result, test.expected)
    }
  }
}

func TestError_data(t *testing.T) {
  tests := []struct {
    err      Error[string]
    expected string
  }{
    {err: Error[string]{Code: -32000, Message: "error", Data: "data"}, expected: "data"},
    {err: Error[string]{Code: -32001, Message: "another error", Data: "more data"}, expected: "more data"},
  }

  for _, test := range tests {
    if result := test.err.data(); result != test.expected {
      t.Errorf("Error(%v).data() = %v; want %v", test.err, result, test.expected)
    }
  }
}

func TestConvertError(t *testing.T) {
  tests := []struct {
    input    error
    expected Error[any]
  }{
    {
      input:    errors.New("standard error"),
      expected: Error[any]{Code: -32000, Message: "standard error", Data: errors.New("standard error")},
    },
    {
      input:    NewError(-32001, "custom error", "data"),
      expected: Error[any]{Code: -32001, Message: "custom error", Data: "data"},
    },
  }

  for _, test := range tests {
    result := convertError(test.input)
    if result.Code != test.expected.Code || result.Message != test.expected.Message || fmt.Sprintf("%v", result.Data) != fmt.Sprintf("%v", test.expected.Data) {
      t.Errorf("convertError(%v) = %v; want %v", test.input, result, test.expected)
    }
  }
}
