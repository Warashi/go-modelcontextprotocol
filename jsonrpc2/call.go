package jsonrpc2

import (
	"context"
	"sync/atomic"
)

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

	return conn.Notify(ctx, method, params)
}
