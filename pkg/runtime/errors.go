package runtime

import "errors"

var (
	// ErrArtifactRequiresBuiltinRuntime indicates compiled artifacts can only run on the builtin runtime.
	ErrArtifactRequiresBuiltinRuntime = errors.New("compiled artifacts require the builtin runtime")

	// ErrHTTPPolicyRequiresBuiltinRuntime indicates HTTP policy options cannot configure a remote runtime.
	ErrHTTPPolicyRequiresBuiltinRuntime = errors.New("HTTP policy options are only supported by the builtin runtime")

	// ErrFSPolicyRequiresBuiltinRuntime indicates filesystem policy options cannot configure a remote runtime.
	ErrFSPolicyRequiresBuiltinRuntime = errors.New("filesystem policy options are only supported by the builtin runtime")

	// ErrDebugRequiresBuiltinRuntime was returned by builtin-only versions of the debugger.
	//
	// Deprecated: Debugging supports builtin and Wire. Unsupported runtimes return errors.ErrUnsupported.
	ErrDebugRequiresBuiltinRuntime = errors.New("debug currently supports only the builtin runtime")
)
