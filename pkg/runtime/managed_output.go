package runtime

import (
	"context"
	"errors"
	"sync"

	"github.com/MontFerret/api"
)

// managedOutput coordinates delivery and releases its owners only after the
// active callback returns. interrupt must unblock I/O without touching chunks.
type managedOutput struct {
	invocation     context.Context
	metadata       api.Metadata
	deliver        func(context.Context, api.Consumer) error
	interrupt      func()
	cleanup        func() error
	mu             sync.Mutex
	state          outputState
	explicit       bool
	cancel         context.CancelFunc
	operation      context.Context
	stopInvocation func() bool
	done           chan struct{}
	cleanupErr     error
}

type outputState uint8

const (
	outputReady outputState = iota
	outputConsuming
	outputStopping
	outputFinalizing
	outputFinalized
)

var _ api.Output = (*managedOutput)(nil)

func newManagedOutput(ctx context.Context, metadata api.Metadata, deliver func(context.Context, api.Consumer) error, interrupt func(), cleanup func() error) *managedOutput {
	o := &managedOutput{invocation: ctx, metadata: metadata, deliver: deliver, interrupt: interrupt, cleanup: cleanup, done: make(chan struct{})}
	o.mu.Lock()
	o.stopInvocation = context.AfterFunc(ctx, func() { o.stop(false) })
	o.mu.Unlock()

	return o
}

func (o *managedOutput) Metadata() api.Metadata { return o.metadata }

func (o *managedOutput) Consume(ctx context.Context, consumer api.Consumer) (err error) {
	if ctx == nil {
		return errors.New("output context is required")
	}

	if consumer == nil {
		return errors.New("output consumer is required")
	}

	o.mu.Lock()
	if err := joinOutputErrors(ctx.Err(), o.invocation.Err()); err != nil {
		o.mu.Unlock()

		return err
	}

	if o.state != outputReady {
		inUse := o.state == outputConsuming && o.operation.Err() == nil
		o.mu.Unlock()
		if inUse {
			return api.ErrOutputInUse
		}

		return api.ErrOutputClosed
	}

	base := ctx
	var deadlineCancel context.CancelFunc
	if deadline, ok := o.invocation.Deadline(); ok {
		if current, present := ctx.Deadline(); !present || deadline.Before(current) {
			base, deadlineCancel = context.WithDeadline(ctx, deadline)
		}
	}

	operation, cancel := context.WithCancel(base)
	o.cancel = cancel
	o.operation = operation
	o.state = outputConsuming
	o.mu.Unlock()
	stop := context.AfterFunc(operation, func() { o.stop(false) })
	defer func() {
		o.mu.Lock()
		err = joinOutputErrors(err, ctx.Err(), o.invocation.Err(), operation.Err())
		if o.explicit {
			err = joinOutputErrors(err, api.ErrOutputClosed)
		}

		o.state = outputFinalizing
		o.mu.Unlock()
		stop()
		cancel()
		if deadlineCancel != nil {
			deadlineCancel()
		}

		o.finalize()
		err = joinOutputErrors(err, o.cleanupErr)
	}()

	return o.deliver(operation, consumer)
}

func (o *managedOutput) Collect(ctx context.Context) (*api.Content, error) {
	var content *api.Content
	err := o.Consume(ctx, func(_ context.Context, data []byte) error {
		if content == nil {
			content = &api.Content{Metadata: o.metadata, Data: []byte{}}
		}

		content.Data = append(content.Data, data...)

		return nil
	})

	return content, err
}

func (o *managedOutput) Close() error {
	o.stop(true)
	<-o.done

	return o.cleanupErr
}

func (o *managedOutput) stop(explicit bool) {
	o.mu.Lock()
	switch o.state {
	case outputReady:
		o.explicit = explicit
		o.state = outputFinalizing
		o.mu.Unlock()
		o.finalize()
	case outputConsuming, outputStopping:
		o.explicit = o.explicit || explicit
		o.state = outputStopping
		cancel := o.cancel
		o.mu.Unlock()
		cancel()
		if o.interrupt != nil {
			o.interrupt()
		}

	default:
		o.mu.Unlock()
	}
}

func (o *managedOutput) finalize() {
	o.stopInvocation()
	err := o.cleanup()
	o.mu.Lock()
	o.cleanupErr = err
	o.state = outputFinalized
	close(o.done)
	o.mu.Unlock()
}
