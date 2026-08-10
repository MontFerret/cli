package discovery

import (
	"context"
	"fmt"
	"sort"

	"golang.org/x/sync/errgroup"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

// Service provides module discovery backed by a registry.
type Service struct {
	registry Registry
}

// New constructs a module discovery service.
func New(registry Registry) *Service {
	return &Service{registry: registry}
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
		idx, sum := index, summary

		group.Go(func() error {
			item, err := s.registry.Module(groupContext, sum.ID)
			if err != nil {
				return fmt.Errorf("load registry module %q: %w", sum.ID, err)
			}

			if len(item.Versions) == 0 {
				return fmt.Errorf("%w: module %q has no versions", barnregistry.ErrMalformedArtifact, item.ID)
			}

			version := sum.Latest
			if version == "" {
				version = item.Versions[0].Version
			}

			results[idx] = SearchResult{Name: item.ID, Version: version, Description: item.Description}

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
