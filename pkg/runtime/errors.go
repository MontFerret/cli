package runtime

import "errors"

var (
	// ErrArtifactRequiresBuiltinRuntime indicates compiled artifacts can only run on the builtin runtime.
	ErrArtifactRequiresBuiltinRuntime = errors.New("compiled artifacts require the builtin runtime")

	// ErrHTTPPolicyRequiresBuiltinRuntime indicates HTTP policy options cannot configure a remote runtime.
	ErrHTTPPolicyRequiresBuiltinRuntime = errors.New("HTTP policy options are only supported by the builtin runtime")

	// ErrFSPolicyRequiresBuiltinRuntime indicates filesystem policy options cannot configure a remote runtime.
	ErrFSPolicyRequiresBuiltinRuntime = errors.New("filesystem policy options are only supported by the builtin runtime")

	// ErrDebugRequiresBuiltinRuntime indicates source debugging is only available
	// through the builtin runtime.
	ErrDebugRequiresBuiltinRuntime = errors.New("debug currently supports only the builtin runtime")
)
