package module

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/MontFerret/cli/v2/pkg/registryclient"
)

// Service coordinates registry discovery, scaffolding, and publication preparation.
type Service struct {
	registry   Registry
	scaffolder *Scaffolder
	publisher  *Publisher
}

// NewService constructs a module lifecycle service.
func NewService(registry Registry, scaffolder *Scaffolder, publisher *Publisher) *Service {
	return &Service{registry: registry, scaffolder: scaffolder, publisher: publisher}
}

// Search returns modules whose identity or description contains query.
func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	catalog, err := s.registry.Catalog(ctx)
	if err != nil {
		return nil, err
	}

	var resultsMu sync.Mutex
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]SearchResult, 0, len(catalog.Modules))
	group, groupContext := errgroup.WithContext(ctx)
	group.SetLimit(6)

	for _, reference := range catalog.Modules {
		ref := reference
		group.Go(func() error {
			item, err := s.registry.Module(groupContext, ref.Href)
			if err != nil {
				return fmt.Errorf("load registry module %q: %w", ref.ID, err)
			}

			if item.ID != ref.ID {
				return fmt.Errorf("%w: catalog module %q resolved to %q", registryclient.ErrMalformed, ref.ID, item.ID)
			}

			if query != "" && !strings.Contains(strings.ToLower(item.ID), query) && !strings.Contains(strings.ToLower(item.Description), query) {
				return nil
			}

			version := item.Latest
			if version == "" {
				version = item.Versions[0].Version
			}

			resultsMu.Lock()
			results = append(results, SearchResult{Name: item.ID, Version: version, Description: item.Description})
			resultsMu.Unlock()

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
	catalog, err := s.registry.Catalog(ctx)
	if err != nil {
		return nil, err
	}

	var href string
	for _, reference := range catalog.Modules {
		if reference.ID == name {
			href = reference.Href

			break
		}
	}

	if href == "" {
		return nil, fmt.Errorf("%w: %s", registryclient.ErrNotFound, name)
	}

	item, err := s.registry.Module(ctx, href)
	if err != nil {
		return nil, err
	}

	if item.ID != name {
		return nil, fmt.Errorf("%w: catalog module %q resolved to %q", registryclient.ErrMalformed, name, item.ID)
	}

	selected := item.Versions[0]
	if item.Latest != "" {
		for _, version := range item.Versions {
			if version.Version == item.Latest {
				selected = version

				break
			}
		}
	}

	version, err := s.registry.Version(ctx, selected.Href)
	if err != nil {
		return nil, err
	}

	if version.ID != name || version.Version != selected.Version {
		return nil, fmt.Errorf("%w: selected version metadata does not match %s@%s", registryclient.ErrMalformed, name, selected.Version)
	}

	versions := make([]string, len(item.Versions))
	for i, available := range item.Versions {
		versions[i] = available.Version
	}

	return &ModuleInfo{
		Name:            item.ID,
		Description:     item.Description,
		License:         item.License,
		Latest:          item.Latest,
		Newest:          item.Versions[0].Version,
		SelectedVersion: selected.Version,
		Versions:        versions,
		Namespace:       version.Namespace,
		Ferret:          version.Ferret,
		Repository:      version.Source.Repository,
		SourcePath:      version.Source.Path,
		Commit:          version.Source.Commit,
		Documentation:   version.Documentation,
	}, nil
}

// Create scaffolds a new module project.
func (s *Service) Create(ctx context.Context, options CreateOptions) (*CreateResult, error) {
	if s.scaffolder == nil {
		return nil, errors.New("module scaffolder is not configured")
	}

	return s.scaffolder.Create(ctx, options)
}

// Publish prepares validated Barn registration records from a local release.
func (s *Service) Publish(ctx context.Context, options PublishOptions) (*Publication, error) {
	if s.publisher == nil {
		return nil, errors.New("module publisher is not configured")
	}

	return s.publisher.Prepare(ctx, options)
}
