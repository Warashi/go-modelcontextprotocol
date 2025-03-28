package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/Warashi/go-modelcontextprotocol/transport"
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

// Handler represents a JSON-RPC 2.0 request handler.
type Handler[Params, Result any] interface {
	HandleRequest(ctx context.Context, req Params) (Result, error)
}

// HandlerFunc represents a JSON-RPC 2.0 request handler function.
type HandlerFunc[Params, Result any] func(ctx context.Context, req Params) (Result, error)

// HandleRequest calls f(ctx, req).
func (f HandlerFunc[Params, Result]) HandleRequest(ctx context.Context, req Params) (Result, error) {
	return f(ctx, req)
}

// RegisterHandler registers a request handler.
func RegisterHandler[Params, Result any](c *Conn, method string, h Handler[Params, Result]) {
	c.handlers[Method(method)] = func(ctx context.Context, req json.RawMessage) (any, error) {
		var r Request[Params]
		if err := json.Unmarshal(req, &r); err != nil {
			return nil, err
		}
		return h.HandleRequest(ctx, r.Params)
	}
}

// ConnectionInitializationOption represents a JSON-RPC 2.0 connection initialization option.
type ConnectionInitializationOption func(*Conn)

// WithHandler registers a request handler.
func WithHandler[Params, Result any](method string, h Handler[Params, Result]) ConnectionInitializationOption {
	return func(c *Conn) {
		RegisterHandler(c, method, h)
	}
}

func WithHandlerFunc[Params, Result any](method string, h HandlerFunc[Params, Result]) ConnectionInitializationOption {
	return func(c *Conn) {
		RegisterHandler(c, method, h)
	}
}

// Conn represents a JSON-RPC 2.0 connection.
type Conn struct {
	transport transport.Session
	mutex     sync.Mutex
	pending   map[ID]chan json.RawMessage
	handlers  map[Method]func(ctx context.Context, req json.RawMessage) (any, error)
	closed    chan struct{}
}

// NewConnection creates a new JSON-RPC 2.0 connection.
func NewConnection(transport transport.Session, opts ...ConnectionInitializationOption) *Conn {
	conn := &Conn{
		transport: transport,
		pending:   make(map[ID]chan json.RawMessage),
		handlers:  make(map[Method]func(ctx context.Context, req json.RawMessage) (any, error)),
		closed:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(conn)
	}

	return conn
}

// Open opens the connection.
// Open returns an error if the connection is already closed.
// Open starts message processing in a new goroutine.
func (c *Conn) Open() error {
	select {
	case <-c.closed:
		return errors.New("connection closed")
	default:
	}

	go c.serve(context.Background())
	return nil
}

// Serve starts serving requests.
// Serve will return an error if the connection is closed.
func (c *Conn) Serve(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errors.New("connection closed")
	default:
	}

	return c.serve(ctx)
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
		return c.transport.Close()
	}
}

// Call sends a request to the server and waits for a response.
func (c *Conn) Call(ctx context.Context, id ID, method string, params any, result any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errors.New("connection closed")
	default:
	}

	req := &Request[any]{
		ID:     id,
		Method: Method(method),
		Params: params,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	respCh := make(chan json.RawMessage, 1)

	c.mutex.Lock()
	c.pending[id] = respCh
	c.mutex.Unlock()

	if err := c.transport.Send(b); err != nil {
		c.mutex.Lock()
		delete(c.pending, id)
		c.mutex.Unlock()
		return err
	}

	select {
	case resp := <-respCh:
		return json.Unmarshal(resp, result)
	case <-ctx.Done():
		c.mutex.Lock()
		delete(c.pending, id)
		c.mutex.Unlock()
		return ctx.Err()
	}
}

// serve starts serving requests.
// serve will return an error if the connection is closed.
func (c *Conn) serve(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errors.New("connection closed")
	default:
	}

	for msg := range c.transport.Receive() {
		// close the connection if the context is done or the connection is closed
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return errors.New("connection closed")
		default:
			// connection is not closed
			// handle the message
			if err := c.handleMessage(ctx, msg); err != nil {
				return err
			}
		}
	}
	// connection is closed
	return errors.New("connection closed")
}

// handleMessage reads a message from the connection and handles it.
func (c *Conn) handleMessage(ctx context.Context, msg json.RawMessage) error {
	trimmedMsg := bytes.TrimSpace(msg)
	if len(trimmedMsg) > 0 && trimmedMsg[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(msg, &batch); err != nil {
			errResp := c.generateErrorResponse(ID{value: nil}, CodeParseError, "Parse error")
			b, _ := json.Marshal(errResp)
			c.mutex.Lock()
			defer c.mutex.Unlock()
			return c.transport.Send(b)
		}
		return c.handleBatchMessage(ctx, batch)
	} else {
		var obj map[string]any
		if err := json.Unmarshal(msg, &obj); err != nil {
			errResp := c.generateErrorResponse(ID{value: nil}, CodeParseError, "Parse error")
			b, _ := json.Marshal(errResp)
			c.mutex.Lock()
			defer c.mutex.Unlock()
			return c.transport.Send(b)
		}
		_ = c.handleRawMessage(ctx, msg)
		return nil
	}
}

// handleRawMessage handles a JSON-RPC 2.0 single message.
func (c *Conn) handleRawMessage(ctx context.Context, msg json.RawMessage) error {
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

// handleRequest handles a JSON-RPC 2.0 request message.
func (c *Conn) handleRequest(ctx context.Context, msg json.RawMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var req Request[json.RawMessage]
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	handler, ok := c.handlers[req.Method]
	if !ok {
		return c.sendError(ctx, req.ID, NewError[any](-32601, "method not found", nil))
	}

	resp, err := handler(ctx, msg)
	if err != nil {
		return c.sendError(ctx, req.ID, err)
	}

	return c.sendResponse(ctx, req.ID, resp)
}

// sendResponse sends a JSON-RPC 2.0 response message.
func (c *Conn) sendResponse(ctx context.Context, id ID, resp any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if id.IsNull() {
		return errors.New("invalid response ID")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	b, err := json.Marshal(&Response[any, any]{
		ID:     id,
		Result: resp,
	})
	if err != nil {
		return err
	}

	if err := c.transport.Send(b); err != nil {
		return err
	}

	return nil
}

// sendError sends a JSON-RPC 2.0 error response message.
func (c *Conn) sendError(ctx context.Context, id ID, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if id.IsNull() {
		return errors.New("invalid error ID")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	b, err := json.Marshal(&Response[any, any]{
		ID:    id,
		Error: convertError(err),
	})
	if err != nil {
		return err
	}

	if err := c.transport.Send(b); err != nil {
		return err
	}

	return nil
}

// handleResponse handles a JSON-RPC 2.0 response message.
func (c *Conn) handleResponse(ctx context.Context, msg json.RawMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var resp Response[json.RawMessage, json.RawMessage]
	if err := json.Unmarshal(msg, &resp); err != nil {
		return err
	}

	var ch chan json.RawMessage
	var ok bool

	c.mutex.Lock()
	ch, ok = c.pending[resp.ID]
	if ok {
		delete(c.pending, resp.ID)
	}
	c.mutex.Unlock()

	if !ok {
		return errors.New("invalid response ID")
	}

	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

	_, err := handler(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

// id manages the request ID.
// The ID is unique and monotonically increasing.
var id atomic.Uint64

// Call sends a request to the server and waits for a response.
// Call returns the result and an error if the request fails.
// When the result is unsuccessful, the error `jsonrpc2.Error[ErrorData]` type.
func Call[Result, ErrorData, Params any](ctx context.Context, conn *Conn, method string, params Params) (Result, error) {
	select {
	case <-ctx.Done():
		var zero Result
		return zero, ctx.Err()
	default:
	}

	id := id.Add(1)

	var result Response[Result, ErrorData]
	if err := conn.Call(ctx, NewID(int(id)), method, params, &result); err != nil {
		return result.Result, err
	}

	return result.tuple()
}

// Notify sends a notification to the server.
// Notify returns an error if the request fails.
func Notify[Params any](ctx context.Context, conn *Conn, method string, params Params) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	req := &Request[Params]{
		Method: Method(method),
		Params: params,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	return conn.transport.Send(b)
}

// generateErrorResponse creates a JSON-RPC 2.0 error response.
func (c *Conn) generateErrorResponse(id ID, code int, message string) *Response[any, any] {
	return &Response[any, any]{
		ID:    id,
		Error: NewError[any](code, message, nil),
	}
}

// handleBatchMessage processes a batch of JSON-RPC 2.0 messages, collects responses for requests and sends a single batch response.
func (c *Conn) handleBatchMessage(ctx context.Context, batch []json.RawMessage) error {
	if len(batch) == 0 {
		errResp := c.generateErrorResponse(ID{value: nil}, CodeInvalidRequest, "Invalid Request")
		b, _ := json.Marshal(errResp)
		c.mutex.Lock()
		defer c.mutex.Unlock()
		return c.transport.Send(b)
	}

	var responses []json.RawMessage

	for _, msg := range batch {
		mType, err := getMessageType(msg)
		if err != nil {
			errResp := c.generateErrorResponse(ID{value: nil}, CodeInvalidRequest, "Invalid Request")
			b, _ := json.Marshal(errResp)
			responses = append(responses, b)
			continue
		}

		switch mType {
		case messageRequest:
			var req Request[json.RawMessage]
			if err := json.Unmarshal(msg, &req); err != nil {
				errResp := c.generateErrorResponse(ID{value: nil}, CodeParseError, "Parse error")
				b, _ := json.Marshal(errResp)
				responses = append(responses, b)
				continue
			}

			handler, ok := c.handlers[req.Method]
			if !ok {
				errResp := c.generateErrorResponse(req.ID, CodeMethodNotFound, "method not found")
				b, _ := json.Marshal(errResp)
				responses = append(responses, b)
				continue
			}

			result, err := handler(ctx, msg)
			if err != nil {
				errResp := c.generateErrorResponse(req.ID, CodeInternalError, err.Error())
				b, _ := json.Marshal(errResp)
				responses = append(responses, b)
				continue
			}

			resp := Response[any, any]{
				ID:     req.ID,
				Result: result,
			}
			b, _ := json.Marshal(resp)
			responses = append(responses, b)
		case messageNotification:
			var req Request[json.RawMessage]
			if err := json.Unmarshal(msg, &req); err != nil {
				continue
			}
			if handler, ok := c.handlers[req.Method]; ok {
				handler(ctx, msg)
			}
			// No response for notifications
		default:
			// Ignore other message types in batch
		}
	}

	if len(responses) > 0 {
		batchResp, _ := json.Marshal(responses)
		c.mutex.Lock()
		defer c.mutex.Unlock()
		return c.transport.Send(batchResp)
	}

	return nil
}
