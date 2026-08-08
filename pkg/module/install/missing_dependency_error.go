package install

import "fmt"

// MissingDependencyError describes a dependency that requires user approval
// before the installer may add it to the project.
type MissingDependencyError struct {
	Path    string
	Version string
}

func (err *MissingDependencyError) Error() string {
	return fmt.Sprintf("project does not select %s; install %s@%s before installing Ferret modules", err.Path, err.Path, err.Version)
}
