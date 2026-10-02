package runtime

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/MontFerret/api"
)

// SourceResources owns one source runtime and the resources created for it.
// Runtime is canonical UAPI for builtin/Wire; Legacy is set only for HTTP Worker
// compatibility. Callers must settle active calls and close any directly created
// children before Close. The exported runtime fields are borrowed by callers.
type SourceResources struct {
	Runtime api.Runtime
	Legacy  Runtime
	// Disconnected closes on loss of an established Wire transport. It is nil
	// for builtin and HTTP. Callers may use it to interrupt idle input, but must
	// still settle runtime calls before closing these resources.
	Disconnected <-chan struct{}
	owners       []io.Closer
	once         sync.Once
	err          error
}

// Close releases the runtime before its native engine or physical transport.
// It is safe to repeat and retains all cleanup failures. Wire performs bounded
// detached logical cleanup even after the execution context has been cancelled.
func (r *SourceResources) Close() error {
	if r == nil {
		return nil
	}

	r.once.Do(func() {
		for _, owner := range r.owners {
			if err := owner.Close(); err != nil {
				r.err = errors.Join(r.err, fmt.Errorf("close source runtime resource: %w", err))
			}
		}
	})

	return r.err
}
