package transport

import (
	"context"
	"encoding/json"
	"io"
	"iter"
)

type Session interface {
	JSONSender
	JSONReceiver
	io.Closer
}

type JSONSender interface {
	Send(v json.RawMessage) error
}

type JSONReceiver interface {
	Receive() iter.Seq[json.RawMessage]
}

type SessionHandler interface {
	HandleSession(context.Context, uint64, Session) error
}

type SessionHandlerFunc func(context.Context, uint64, Session) error

func (f SessionHandlerFunc) HandleSession(ctx context.Context, id uint64, s Session) error {
	return f(ctx, id, s)
}
