package publish

import (
	"context"
	"path"
	"path/filepath"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	barnregistry "github.com/MontFerret/barn/pkg/registry"
	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

// Publisher adapts the CLI's publication options to Barn's public API.
type Publisher struct {
	registry *barnregistry.Client
	prepare  func(context.Context, barnpublish.Request) (*barnpublish.Result, error)
}

// New constructs a publication preparer.
func New(registry *barnregistry.Client) *Publisher {
	return &Publisher{registry: registry, prepare: barnpublish.Prepare}
}

// Prepare derives the CLI's optional default tag and delegates validation to Barn.
func (p *Publisher) Prepare(ctx context.Context, options Options) (*barnpublish.Result, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}

	tag := options.Tag
	if tag == "" {
		manifest, err := modulemanifest.LoadFile(filepath.Join(directory, modulemanifest.ManifestFilename))
		if err != nil {
			return nil, err
		}

		tag = "v" + manifest.Version
		if manifest.Repository != nil && manifest.Repository.Directory != "" {
			tag = path.Join(manifest.Repository.Directory, tag)
		}
	}

	return p.prepare(ctx, barnpublish.Request{Directory: directory, Tag: tag, Registry: p.registry})
}
