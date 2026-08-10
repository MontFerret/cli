package publish

import (
	"path"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

func defaultTag(manifest *modulemanifest.Manifest) string {
	tag := "v" + manifest.Version

	if manifest.Repository != nil && manifest.Repository.Directory != "" {
		tag = path.Join(manifest.Repository.Directory, tag)
	}

	return tag
}
