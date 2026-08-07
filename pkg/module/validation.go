package module

import modulemanifest "github.com/MontFerret/specs/pkg/module"

// ValidateManifest loads and validates a Ferret module manifest.
func ValidateManifest(path string) (*modulemanifest.Manifest, error) {
	return modulemanifest.LoadFile(path)
}
