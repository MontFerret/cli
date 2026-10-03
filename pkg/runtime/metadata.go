package runtime

// EmbeddedVersion returns CLI-injected metadata for the embedded native engine.
// It does not describe an external runtime.
func EmbeddedVersion() string {
	return version
}
