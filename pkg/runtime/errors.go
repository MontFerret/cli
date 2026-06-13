package runtime

import "errors"

// ErrArtifactRequiresBuiltinRuntime indicates compiled artifacts can only run on the builtin runtime.
var ErrArtifactRequiresBuiltinRuntime = errors.New("compiled artifacts require the builtin runtime")

// DebugRequiresBuiltinRuntimeError reports an attempt to debug through a
// runtime that cannot create local debugger sessions.
type DebugRequiresBuiltinRuntimeError struct{}

func (*DebugRequiresBuiltinRuntimeError) Error() string {
	return "debugging requires the builtin runtime"
}

// ErrDebugRequiresBuiltinRuntime indicates source debugging is only available
// through the builtin runtime.
var ErrDebugRequiresBuiltinRuntime = &DebugRequiresBuiltinRuntimeError{}
