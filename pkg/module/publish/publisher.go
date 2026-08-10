package publish

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	barnregistry "github.com/MontFerret/barn/pkg/registry"
	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

// Publisher adapts the CLI's publication options to Barn's public API.
type Publisher struct {
	registry  *barnregistry.Client
	prepare   func(context.Context, barnpublish.Request) (*barnpublish.Result, error)
	submitter Submitter
}

// New constructs a publication workflow.
func New(registry *barnregistry.Client, submitters ...Submitter) *Publisher {
	var submitter Submitter

	if len(submitters) > 0 {
		submitter = submitters[0]
	}

	return &Publisher{registry: registry, prepare: barnpublish.Prepare, submitter: submitter}
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

		tag = defaultTag(manifest)
	}

	return p.prepare(ctx, barnpublish.Request{Directory: directory, Tag: tag, Registry: p.registry})
}

// Publish prepares a release and submits it unless a non-mutating mode is selected.
// When submission fails, the returned result retains the successfully prepared records.
func (p *Publisher) Publish(ctx context.Context, options Options) (*Result, error) {
	mode := options.Mode
	if mode == "" {
		mode = ModeSubmit
	}

	if mode != ModeSubmit && mode != ModeDryRun && mode != ModePrint {
		return nil, fmt.Errorf("unsupported publication mode %q", mode)
	}

	request, manifest, err := p.request(options)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Status:  StatusReady,
		Module:  manifest.Name,
		Version: manifest.Version,
		Tag:     request.Tag,
	}

	prepared, err := p.prepare(ctx, request)
	if err != nil {
		if errors.Is(err, barnpublish.ErrVersionAlreadyPublished) {
			result.Status = StatusAlreadyPublished

			return result, nil
		}

		return nil, err
	}

	result.Prepared = prepared

	if mode != ModeSubmit {
		return result, nil
	}

	if p.submitter == nil {
		return result, errors.New("GitHub Registry submission is not configured")
	}

	submission, err := p.submitter.Submit(ctx, prepared)
	if err != nil {
		return result, err
	}

	if submission == nil {
		return result, errors.New("GitHub Registry submission returned no result")
	}

	if submission.AlreadyPublished {
		result.Status = StatusAlreadyPublished

		return result, nil
	}

	if submission.URL == "" {
		return result, errors.New("GitHub Registry submission returned no pull request URL")
	}

	result.PullRequestURL = submission.URL

	if submission.Existing {
		result.Status = StatusExistingSubmission
	} else {
		result.Status = StatusSubmitted
	}

	return result, nil
}

func (p *Publisher) request(options Options) (barnpublish.Request, *modulemanifest.Manifest, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}

	manifest, err := modulemanifest.LoadFile(filepath.Join(directory, modulemanifest.ManifestFilename))
	if err != nil {
		return barnpublish.Request{}, nil, err
	}

	tag := options.Tag
	if tag == "" {
		tag = defaultTag(manifest)
	}

	return barnpublish.Request{Directory: directory, Tag: tag, Registry: p.registry}, manifest, nil
}
