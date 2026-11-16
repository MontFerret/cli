// Package testutil provides test inputs shared by interactive commands.
package testutil

import "context"

// ContextReader can be passed to StartWithIO as a cooperative borrowed reader.
type ContextReader func(context.Context, []byte) (int, error)

func (r ContextReader) Read(b []byte) (int, error) {
	return r(context.Background(), b)
}

func (r ContextReader) ReadContext(ctx context.Context, b []byte) (int, error) {
	return r(ctx, b)
}
