package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/MontFerret/api"
)

// Resources owns the selected runtime and the resources created for it.
// Runtime is the borrowed canonical UAPI for builtin, Wire, or HTTP Worker.
// Callers must settle active outputs and close any directly created children
// before Close.
type Resources struct {
	Runtime api.Runtime
	// Disconnected closes on unexpected loss of the Wire Connect RPC. It is nil
	// for builtin and HTTP. Callers may use it to interrupt idle input, but must
	// still settle runtime calls before closing these resources.
	Disconnected <-chan struct{}
	builtin      *Builtin
	owners       []io.Closer
	once         sync.Once
	closing      atomic.Bool
	err          error
}

// Close releases the runtime before its native engine or physical transport.
// It is safe to repeat and retains all cleanup failures. Wire performs bounded
// detached logical cleanup even after the execution context has been cancelled.
func (r *Resources) Close() error {
	if r == nil {
		return nil
	}

	r.once.Do(func() {
		r.closing.Store(true)

		for _, owner := range r.owners {
			if err := owner.Close(); err != nil {
				r.err = errors.Join(r.err, fmt.Errorf("close runtime resource: %w", err))
			}
		}
	})

	return r.err
}

// CommandContext interrupts interactive input and execution when the Wire connection ends.
// The caller must stop the watcher after active commands settle.
func (r *Resources) CommandContext(parent context.Context, endpoint string) (context.Context, context.CancelFunc) {
	if r.Disconnected == nil {
		return parent, func() {}
	}

	ctx, cancel := context.WithCancelCause(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)

		select {
		case <-ctx.Done():
		case <-r.Disconnected:
			cancel(fmt.Errorf("wire runtime %s disconnected; check the runtime host", endpoint))
		}
	}()

	return ctx, func() {
		cancel(nil)
		<-done
	}
}
