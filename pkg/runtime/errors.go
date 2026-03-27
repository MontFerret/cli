package runtime

import "errors"

// ErrArtifactRequiresBuiltinRuntime indicates compiled artifacts can only run on the builtin runtime.
var ErrArtifactRequiresBuiltinRuntime = errors.New("compiled artifacts require the builtin runtime")
