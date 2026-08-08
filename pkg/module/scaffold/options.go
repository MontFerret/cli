package scaffold

import "fmt"

// Options controls module project scaffolding.
type Options struct {
	Name      string
	GoModule  string
	Directory string
	Namespace string
}

// DefaultOptions derives the editable defaults used by the guided init flow.
func DefaultOptions(name string) (Options, error) {
	leaf, err := moduleLeaf(name)
	if err != nil {
		return Options{}, err
	}

	owner := name[:len(name)-len(leaf)-1]
	options := Options{
		Name:      name,
		GoModule:  fmt.Sprintf("github.com/%s/ferret-%s", owner, leaf),
		Directory: leaf,
		Namespace: namespaceIdentifier(leaf),
	}

	if _, err := validateAndResolveOptions(options); err != nil {
		return Options{}, err
	}

	return options, nil
}

// Validate checks scaffold input without reading or writing the filesystem.
func (options Options) Validate() error {
	_, err := validateAndResolveOptions(options)
	return err
}
