package jsonrpc2

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Method represents a JSON-RPC 2.0 request method.
type Method string

// ID represents a JSON-RPC 2.0 request ID.
// The ID can be a string, number, or null.
// If the ID is a string, it must be unique.
// If the ID is a number, it must be an integer.
// If the ID is null, the request is a notification.
type ID struct {
	value any
}

// NewID creates a new ID from a string or an integer.
func NewID[T interface{ string | int }](v T) ID {
	return ID{value: v}
}

// IsNull returns true if the ID is null.
func (id ID) IsNull() bool {
	return id.value == nil
}

// String returns the ID as a string.
// If the ID is an integer, it is converted to a string.
// If the ID is null, it returns an empty string.
func (id ID) String() string {
	switch v := id.value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case nil:
		return ""
	default:
		panic("invalid ID type")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (id *ID) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		id.value = v
	case float64:
		id.value = int(v)
	case nil:
		id.value = v
	default:
		return fmt.Errorf("invalid ID type: %T", v)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (id ID) MarshalJSON() ([]byte, error) {
	switch v := id.value.(type) {
	case string:
		return json.Marshal(v)
	case int:
		return json.Marshal(v)
	case nil:
		return json.Marshal(v)
	default:
		return nil, errors.New("invalid ID type")
	}
}

// Request represents a JSON-RPC 2.0 request.
// The request object must contain a method.
// The request object may contain an ID.
// The request object may contain parameters.
type Request[Params any] struct {
	ID     ID     `json:"id,omitzero"`
	Method Method `json:"method"`
	Params Params `json:"params,omitempty,omitzero"`
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Request[Params]) MarshalJSON() ([]byte, error) {
	if r.ID.IsNull() {
		return json.Marshal(struct {
			JSONRPC string `json:"jsonrpc"`
			Method  Method `json:"method"`
			Params  Params `json:"params,omitempty,omitzero"`
		}{
			JSONRPC: "2.0",
			Method:  r.Method,
			Params:  r.Params,
		})
	}
	return json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      ID     `json:"id,omitzero"`
		Method  Method `json:"method"`
		Params  Params `json:"params,omitempty,omitzero"`
	}{
		JSONRPC: "2.0",
		ID:      r.ID,
		Method:  r.Method,
		Params:  r.Params,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Request[Params]) UnmarshalJSON(data []byte) error {
	var req struct {
		JSONRPC string `json:"jsonrpc"`
		ID      ID     `json:"id,omitzero"`
		Method  Method `json:"method"`
		Params  Params `json:"params,omitempty,omitzero"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	if req.JSONRPC != "2.0" {
		return errors.New("invalid JSON-RPC version")
	}

	r.ID = req.ID
	r.Method = req.Method
	r.Params = req.Params
	return nil
}

// Response represents a JSON-RPC 2.0 response.
// The response object must contain a unique ID.
// The response object may contain a result or an error.
type Response[Result, ErrorData any] struct {
	ID     ID               `json:"id,omitzero"`
	Result Result           `json:"result,omitempty,omitzero"`
	Error  Error[ErrorData] `json:"error,omitzero"`
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Response[Result, ErrorData]) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		JSONRPC string           `json:"jsonrpc"`
		ID      ID               `json:"id,omitzero"`
		Result  Result           `json:"result,omitempty,omitzero"`
		Error   Error[ErrorData] `json:"error,omitzero"`
	}{
		JSONRPC: "2.0",
		ID:      r.ID,
		Result:  r.Result,
		Error:   r.Error,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Response[Result, ErrorData]) UnmarshalJSON(data []byte) error {
	var resp struct {
		JSONRPC string           `json:"jsonrpc"`
		ID      ID               `json:"id,omitzero"`
		Result  Result           `json:"result,omitempty,omitzero"`
		Error   Error[ErrorData] `json:"error,omitzero"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if resp.JSONRPC != "2.0" {
		return errors.New("invalid JSON-RPC version")
	}

	r.ID = resp.ID
	r.Result = resp.Result
	r.Error = resp.Error
	return nil
}

// tuple returns the result and an error if the response is unsuccessful.
func (r *Response[Result, ErrorData]) tuple() (Result, error) {
	if r.Error.Code != 0 {
		return r.Result, r.Error
	}
	return r.Result, nil
}

// Error represents a JSON-RPC 2.0 error.
// The error object must contain a code and a message.
// The error object may contain data.
type Error[Data any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Data   `json:"data,omitempty,omitzero"`
}

// NewError creates a new JSON-RPC 2.0 error.
func NewError[Data any](code int, message string, data Data) Error[Data] {
	return Error[Data]{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Error implements the error interface.
func (e Error[Data]) Error() string {
	return e.Message
}

// code returns the error code.
func (e Error[Data]) code() int {
	return e.Code
}

// message returns the error message.
func (e Error[Data]) message() string {
	return e.Message
}

// data returns the error data as an any.
func (e Error[Data]) data() any {
	return e.Data
}

// convertError converts an error to a JSON-RPC 2.0 error.
func convertError(err error) Error[any] {
	if err == nil {
		panic("nil error")
	}

	if e, ok := err.(interface {
		code() int
		message() string
		data() any
	}); ok {
		return Error[any]{
			Code:    e.code(),
			Message: e.message(),
			Data:    e.data(),
		}
	}

	return Error[any]{
		Code:    -32000,
		Message: err.Error(),
		Data:    err,
	}
}

// messageType represents a JSON-RPC 2.0 message type.
type messageType int

const (
	_ messageType = iota
	// messageRequest represents a JSON-RPC 2.0 request message.
	messageRequest
	// messageResponse represents a JSON-RPC 2.0 response message.
	messageResponse
	// messageNotification represents a JSON-RPC 2.0 notification message.
	messageNotification
)

// getMessageType returns the message type of a JSON-RPC 2.0 message.
func getMessageType(msg json.RawMessage) (messageType, error) {
	var v map[string]any
	if err := json.Unmarshal(msg, &v); err != nil {
		return 0, err
	}

	if v, ok := v["jsonrpc"]; !ok || v != "2.0" {
		return 0, errors.New("invalid JSON-RPC version")
	}

	if _, ok := v["error"]; ok {
		// if error is present, it's a response
		// error response may not have an id
		return messageResponse, nil
	}

	if _, ok := v["result"]; ok {
		if _, ok := v["id"]; ok {
			// if result and id are present, it's a response
			return messageResponse, nil
		}
		// if result present, but id is missing, it's invalid
		return 0, errors.New("invalid message type")
	}

	if _, ok := v["method"]; ok {
		if _, ok := v["id"]; ok {
			// if method and id are present, it's a request
			return messageRequest, nil
		}
		// if method is present, but id is missing, it's a notification
		return messageNotification, nil
	}

	// otherwise, it's invalid
	return 0, errors.New("invalid message type")
}
