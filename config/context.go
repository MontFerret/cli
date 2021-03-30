package config

import "context"

var ctxKey = struct{}{}

func With(ctx context.Context, store *Store) context.Context {
	return context.WithValue(ctx, ctxKey, store)
}

func From(ctx context.Context) *Store {
	val := ctx.Value(ctxKey)

	store, ok := val.(*Store)

	if ok {
		return store
	}

	return nil
}
