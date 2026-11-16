package diagnostics

import (
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	native "github.com/MontFerret/ferret/v2/pkg/debugger"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

// BreakpointNotFound distinguishes an unknown ID from a missing debugger session.
func BreakpointNotFound(err error) bool {
	var remote *client.Error

	return errors.Is(err, runtime.ErrNotFound) ||
		(errors.As(err, &remote) && remote.Category == failure.CategoryBreakpointNotFound)
}

// DebuggerUnavailable recognizes a lost session or transport. A rejected
// inspection or pause is an ordinary command error and leaves the stop usable.
func DebuggerUnavailable(err error) bool {
	if errors.Is(err, client.ErrClosed) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	var state *native.StateError
	if errors.As(err, &state) && state.State == "closed" {
		return true
	}

	var rpc *client.Error
	if errors.As(err, &rpc) {
		switch rpc.Category {
		case failure.CategoryConnectionNotFound, failure.CategoryPlanNotFound,
			failure.CategoryDebugSessionNotFound, failure.CategorySessionNotFound, failure.CategoryInternalRuntime:
			return true
		}

		return status.Code(rpc) == codes.Unavailable || status.Code(rpc) == codes.DataLoss
	}

	return false
}
