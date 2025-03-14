package jsonrpc2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
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
		return ""
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
			Method Method `json:"method"`
			Params Params `json:"params,omitempty,omitzero"`
		}{
			Method: r.Method,
			Params: r.Params,
		})
	}
	return json.Marshal(struct {
		ID     ID     `json:"id,omitempty,omitzero"`
		Method Method `json:"method"`
		Params Params `json:"params,omitempty,omitzero"`
	}{
		ID:     r.ID,
		Method: r.Method,
		Params: r.Params,
	})
}

// Response represents a JSON-RPC 2.0 response.
// The response object must contain a unique ID.
// The response object may contain a result or an error.
type Response[Result, ErrorData any] struct {
	ID     ID                `json:"id,omitempty,omitzero"`
	Result *Result           `json:"result,omitempty,omitzero"`
	Error  *Error[ErrorData] `json:"error,omitempty,omitzero"`
}

// Error represents a JSON-RPC 2.0 error.
// The error object must contain a code and a message.
// The error object may contain data.
type Error[Data any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Data   `json:"data,omitempty,omitzero"`
}

// Error implements the error interface.
func (e *Error[Data]) Error() string {
	return e.Message
}

// code returns the error code.
func (e *Error[Data]) code() int {
	return e.Code
}

// message returns the error message.
func (e *Error[Data]) message() string {
	return e.Message
}

// data returns the error data as an any.
func (e *Error[Data]) data() any {
	return e.Data
}

// convertError converts an error to a JSON-RPC 2.0 error.
func convertError(err error) *Error[any] {
	if err == nil {
		panic("nil error")
	}

	if e, ok := err.(interface {
		code() int
		message() string
		data() any
	}); ok {
		return &Error[any]{
			Code:    e.code(),
			Message: e.message(),
			Data:    e.data(),
		}
	}

	return &Error[any]{
		Code:    -32000,
		Message: err.Error(),
		Data:    err,
	}
}

// Handler represents a JSON-RPC 2.0 request handler.
type Handler interface {
	HandleRequest(ctx context.Context, req json.RawMessage) (any, error)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as JSON-RPC 2.0 request handlers.
type HandlerFunc func(ctx context.Context, req json.RawMessage) (any, error)

// HandleRequest calls f(ctx, req).
func (f HandlerFunc) HandleRequest(ctx context.Context, req json.RawMessage) (any, error) {
	return f(ctx, req)
}

// Conn represents a JSON-RPC 2.0 connection.
type Conn struct {
	reader   io.Reader
	writer   io.Writer
	enc      *json.Encoder
	dec      *json.Decoder
	mutex    sync.Mutex
	pending  map[ID]chan json.RawMessage
	handlers map[Method]Handler
	closed   chan struct{}
}

// NewConnection creates a new JSON-RPC 2.0 connection.
func NewConnection(r io.Reader, w io.Writer) *Conn {
	conn := &Conn{
		enc:      json.NewEncoder(w),
		dec:      json.NewDecoder(r),
		pending:  make(map[ID]chan json.RawMessage),
		handlers: make(map[Method]Handler),
		closed:   make(chan struct{}),
	}

	go conn.serve(context.Background())

	return conn
}

// Close closes the connection.
// Any blocked Call will return with an error.
// It is safe to call Close multiple times.
// Close will return nil if the connection is already closed.
// Close will return an error if the connection is not closed cleanly.
// Close closes the underlying reader and writer if they implement io.Closer.
func (c *Conn) Close() error {
	select {
	case _, ok := <-c.closed:
		if !ok {
			// Already closed
			return nil
		}
		panic("unreachable")
	default:
		// Not closed yet
		// Close the connection
		close(c.closed)
		var err error
		if closer, ok := c.writer.(io.Closer); ok {
			err = errors.Join(err, closer.Close())
		}
		if closer, ok := c.reader.(io.Closer); ok {
			err = errors.Join(err, closer.Close())
		}
		return err
	}
}

// Call sends a request to the server and waits for a response.
func (c *Conn) Call(ctx context.Context, id ID, method string, params any, result any) error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	if !id.IsNull() {
		req["id"] = id
	}

	respCh := make(chan json.RawMessage, 1)

	c.mutex.Lock()
	c.pending[id] = respCh
	if err := c.enc.Encode(req); err != nil {
		delete(c.pending, id)
		c.mutex.Unlock()
		return err
	}
	c.mutex.Unlock()

	select {
	case resp := <-respCh:
		return json.Unmarshal(resp, result)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// serve starts serving requests.
// serve will return an error if the connection is closed.
func (c *Conn) serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return errors.New("connection closed")
		default:
			go c.handleMessage(ctx)
		}
	}
}

// handleMessage reads a message from the connection and handles it.
func (c *Conn) handleMessage(ctx context.Context) error {
	var msg json.RawMessage
	if err := c.dec.Decode(&msg); err != nil {
		return err
	}

	switch t, err := getMessageType(msg); t {
	case messageRequest:
		return c.handleRequest(ctx, msg)
	case messageResponse:
		return c.handleResponse(ctx, msg)
	case messageNotification:
		return c.handleNotification(ctx, msg)
	default:
		if err != nil {
			return err
		}
		return errors.New("invalid message type")
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

	if _, ok := v["id"]; !ok {
		return messageNotification, nil
	}
	if _, ok := v["method"]; ok {
		return messageRequest, nil
	}
	if _, ok := v["result"]; ok {
		return messageResponse, nil
	}
	if _, ok := v["error"]; ok {
		return messageResponse, nil
	}

	return 0, errors.New("invalid message type")
}

// handleRequest handles a JSON-RPC 2.0 request message.
func (c *Conn) handleRequest(ctx context.Context, msg json.RawMessage) error {
	var req Request[json.RawMessage]
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	handler, ok := c.handlers[req.Method]
	if !ok {
		return errors.New("method not found")
	}

	resp, err := handler.HandleRequest(ctx, req.Params)
	if err != nil {
		return c.sendError(ctx, req.ID, err)
	}

	return c.sendResponse(ctx, req.ID, resp)
}

// sendResponse sends a JSON-RPC 2.0 response message.
func (c *Conn) sendResponse(ctx context.Context, id ID, resp any) error {
	if id.IsNull() {
		return errors.New("invalid response ID")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.enc.Encode(Response[any, any]{
		ID:     id,
		Result: &resp,
	}); err != nil {
		return err
	}

	return nil
}

// sendError sends a JSON-RPC 2.0 error response message.
func (c *Conn) sendError(ctx context.Context, id ID, err error) error {
	if id.IsNull() {
		return errors.New("invalid error ID")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.enc.Encode(Response[any, any]{
		ID:    id,
		Error: convertError(err),
	}); err != nil {
		return err
	}

	return nil
}

// handleResponse handles a JSON-RPC 2.0 response message.
func (c *Conn) handleResponse(ctx context.Context, msg json.RawMessage) error {
	var resp Response[json.RawMessage, json.RawMessage]
	if err := json.Unmarshal(msg, &resp); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	ch, ok := c.pending[resp.ID]
	if !ok {
		return errors.New("invalid response ID")
	}

	ch <- msg
	delete(c.pending, resp.ID)

	return nil
}

// handleNotification handles a JSON-RPC 2.0 notification message.
func (c *Conn) handleNotification(ctx context.Context, msg json.RawMessage) error {
	var req Request[json.RawMessage]
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	handler, ok := c.handlers[req.Method]
	if !ok {
		return errors.New("method not found")
	}

	_, err := handler.HandleRequest(ctx, req.Params)
	if err != nil {
		return err
	}

	return nil
}
