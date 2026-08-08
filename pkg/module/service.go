package module

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"golang.org/x/sync/errgroup"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

// Service coordinates registry discovery, application installation, scaffolding, and publication preparation.
type Service struct {
	registry   Registry
	scaffolder *Scaffolder
	publisher  PublicationPreparer
	installer  *Installer
}

// NewService constructs a module lifecycle service.
func NewService(registry Registry, scaffolder *Scaffolder, publisher PublicationPreparer) *Service {
	return &Service{
		registry: registry, scaffolder: scaffolder, publisher: publisher,
		installer: NewInstaller(registry, nil),
	}
}

// Search returns modules whose canonical identity or description contains query.
func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	summaries, err := s.registry.Search(ctx, barnregistry.SearchOptions{Query: query})
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(summaries))
	group, groupContext := errgroup.WithContext(ctx)
	group.SetLimit(6)

	for index, summary := range summaries {
		index, summary := index, summary
		group.Go(func() error {
			item, err := s.registry.Module(groupContext, summary.ID)
			if err != nil {
				return fmt.Errorf("load registry module %q: %w", summary.ID, err)
			}

			if len(item.Versions) == 0 {
				return fmt.Errorf("%w: module %q has no versions", barnregistry.ErrMalformedArtifact, item.ID)
			}

			version := summary.Latest
			if version == "" {
				version = item.Versions[0].Version
			}

			results[index] = SearchResult{Name: item.ID, Version: version, Description: item.Description}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	return results, nil
}

// Info returns detailed metadata for one registry module.
func (s *Service) Info(ctx context.Context, name string) (*ModuleInfo, error) {
	item, err := s.registry.Module(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(item.Versions) == 0 {
		return nil, fmt.Errorf("%w: module %q has no versions", barnregistry.ErrMalformedArtifact, name)
	}

	selected := item.Versions[0].Version
	if item.Latest != "" {
		selected = item.Latest
	}

	version, err := s.registry.Version(ctx, name, selected)
	if err != nil {
		return nil, err
	}

	versions := make([]string, len(item.Versions))
	for i, available := range item.Versions {
		versions[i] = available.Version
	}

	return &ModuleInfo{
		Name:            item.ID,
		Description:     item.Description,
		Latest:          item.Latest,
		Newest:          item.Versions[0].Version,
		SelectedVersion: selected,
		Versions:        versions,
		Namespace:       version.Namespace,
		Ferret:          version.Ferret,
		Repository:      version.Source.Repository,
		SourcePath:      version.Source.Path,
		Commit:          version.Source.Commit,
		Documentation:   version.Content["documentation"],
	}, nil
}

// Install adds a registered module to an existing Go application.
func (s *Service) Install(ctx context.Context, options InstallOptions) (*InstallResult, error) {
	installer := s.installer
	if installer == nil {
		installer = NewInstaller(s.registry, nil)
	}

	return installer.Install(ctx, options)
}

// Create scaffolds a new module project.
func (s *Service) Create(ctx context.Context, options CreateOptions) (*CreateResult, error) {
	if s.scaffolder == nil {
		return nil, errors.New("module scaffolder is not configured")
	}

	return s.scaffolder.Create(ctx, options)
}

// Publish prepares validated Barn registration records from a local release.
func (s *Service) Publish(ctx context.Context, options PublishOptions) (*barnpublish.Result, error) {
	if s.publisher == nil {
		return nil, errors.New("module publisher is not configured")
	}

	return s.publisher.Prepare(ctx, options)
}
