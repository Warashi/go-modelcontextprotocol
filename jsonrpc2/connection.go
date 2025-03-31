package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	"github.com/Warashi/go-modelcontextprotocol/transport"
)

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

// WithHandlerFunc registers a request handler function.
func WithHandlerFunc[Params, Result any](method string, h HandlerFunc[Params, Result]) ConnectionInitializationOption {
	return func(c *Conn) {
		RegisterHandler(c, method, h)
	}
}

// WithLogger sets a logger for the connection.
func WithLogger(logger *slog.Logger) ConnectionInitializationOption {
	return func(c *Conn) {
		c.logger = logger
	}
}

// Conn represents a JSON-RPC 2.0 connection.
type Conn struct {
	transport transport.Session
	mutex     sync.Mutex
	pending   map[ID]chan json.RawMessage
	handlers  map[Method]func(ctx context.Context, req json.RawMessage) (any, error)
	closed    chan struct{}
	logger    *slog.Logger
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

// log logs a message using the connection's logger.
func (c *Conn) log(msg string, args ...any) {
	if c.logger != nil {
		c.logger.Debug(msg, args...)
	}
}

// handleRequest handles a JSON-RPC 2.0 request message.
func (c *Conn) handleRequest(ctx context.Context, msg json.RawMessage) error {
	c.log("handleRequest", slog.String("message", string(msg)))

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

	c.log("sendResponse", slog.String("body", string(b)))

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

	c.log("sendError", slog.String("body", string(b)))

	if err := c.transport.Send(b); err != nil {
		return err
	}

	return nil
}

// handleResponse handles a JSON-RPC 2.0 response message.
func (c *Conn) handleResponse(ctx context.Context, msg json.RawMessage) error {
	c.log("handleResponse", slog.String("message", string(msg)))

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
