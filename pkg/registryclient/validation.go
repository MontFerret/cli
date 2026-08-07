package registryclient

import "fmt"

func validateRootDocument(document *rootDocument) error {
	if document.SchemaVersion != 1 {
		return fmt.Errorf("%w: unsupported root schemaVersion %d", ErrMalformed, document.SchemaVersion)
	}

	if document.Artifacts["modules"] == "" {
		return fmt.Errorf("%w: root index does not declare the modules artifact", ErrMalformed)
	}

	return nil
}

func validateCatalogDocument(document *catalogDocument) error {
	if document.SchemaVersion != 1 {
		return fmt.Errorf("%w: unsupported module catalog schemaVersion %d", ErrMalformed, document.SchemaVersion)
	}

	if document.Modules == nil {
		return fmt.Errorf("%w: module catalog does not contain a modules array", ErrMalformed)
	}

	seen := make(map[string]struct{}, len(document.Modules))
	for _, item := range document.Modules {
		if item.ID == "" || item.Href == "" {
			return fmt.Errorf("%w: module catalog entry requires id and href", ErrMalformed)
		}

		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("%w: module catalog contains duplicate id %q", ErrMalformed, item.ID)
		}

		seen[item.ID] = struct{}{}
	}

	return nil
}

func validateModuleDocument(document *moduleDocument) error {
	if document.SchemaVersion != 1 {
		return fmt.Errorf("%w: unsupported module schemaVersion %d", ErrMalformed, document.SchemaVersion)
	}

	if document.ID == "" || document.Owner == "" || document.Name == "" || document.Description == "" {
		return fmt.Errorf("%w: module document requires id, owner, name, and description", ErrMalformed)
	}

	if document.ID != document.Owner+"/"+document.Name {
		return fmt.Errorf("%w: module id %q does not match owner/name", ErrMalformed, document.ID)
	}

	if len(document.Versions) == 0 {
		return fmt.Errorf("%w: module %q has no versions", ErrMalformed, document.ID)
	}

	foundLatest := document.Latest == ""
	seen := make(map[string]struct{}, len(document.Versions))
	for _, item := range document.Versions {
		if item.Version == "" || item.Href == "" {
			return fmt.Errorf("%w: module version entry requires version and href", ErrMalformed)
		}

		if _, exists := seen[item.Version]; exists {
			return fmt.Errorf("%w: module %q contains duplicate version %q", ErrMalformed, document.ID, item.Version)
		}

		seen[item.Version] = struct{}{}
		if item.Version == document.Latest {
			foundLatest = true
		}
	}

	if !foundLatest {
		return fmt.Errorf("%w: module latest version %q is not in versions", ErrMalformed, document.Latest)
	}

	return nil
}

func validateVersionDocument(document *versionDocument) error {
	if document.SchemaVersion != 1 {
		return fmt.Errorf("%w: unsupported version schemaVersion %d", ErrMalformed, document.SchemaVersion)
	}

	if document.ID == "" || document.Version == "" || document.Namespace == "" {
		return fmt.Errorf("%w: version document requires id, version, and namespace", ErrMalformed)
	}

	if document.Source.Repository == "" || document.Source.Commit == "" {
		return fmt.Errorf("%w: version source requires repository and commit", ErrMalformed)
	}

	if document.Content["documentation"] == "" {
		return fmt.Errorf("%w: version content does not declare documentation", ErrMalformed)
	}

	return nil
}
