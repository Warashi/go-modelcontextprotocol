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
	HandleSession(context.Context, Session) (id uint64, err error)
}

type SessionHandlerFunc func(context.Context, Session) (id uint64, err error)

func (f SessionHandlerFunc) HandleSession(ctx context.Context, s Session) (id uint64, err error) {
	return f(ctx, s)
}
