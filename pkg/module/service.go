package module

import (
	"context"
	"errors"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

// Service coordinates module discovery, installation, scaffolding, and publication.
type Service struct {
	discovery  Discovery
	installer  Installer
	scaffolder Scaffolder
	publisher  Publisher
}

// NewService constructs a module lifecycle service from its workflow components.
func NewService(discovery Discovery, installer Installer, scaffolder Scaffolder, publisher Publisher) *Service {
	return &Service{
		discovery: discovery, installer: installer, scaffolder: scaffolder, publisher: publisher,
	}
}

// Search returns registered modules matching query.
func (s *Service) Search(ctx context.Context, query string) ([]discovery.SearchResult, error) {
	if s.discovery == nil {
		return nil, errors.New("module discovery is not configured")
	}

	return s.discovery.Search(ctx, query)
}

// Info returns detailed metadata for one registered module.
func (s *Service) Info(ctx context.Context, name string) (*discovery.ModuleInfo, error) {
	if s.discovery == nil {
		return nil, errors.New("module discovery is not configured")
	}

	return s.discovery.Info(ctx, name)
}

// Install adds a registered module to an existing Go application.
func (s *Service) Install(ctx context.Context, options install.Options) (*install.Result, error) {
	if s.installer == nil {
		return nil, errors.New("module installer is not configured")
	}

	return s.installer.Install(ctx, options)
}

// Create scaffolds a new module project.
func (s *Service) Create(ctx context.Context, options scaffold.Options) (*scaffold.Result, error) {
	if s.scaffolder == nil {
		return nil, errors.New("module scaffolder is not configured")
	}

	return s.scaffolder.Create(ctx, options)
}

// Publish prepares validated Barn registration records from a local release.
func (s *Service) Publish(ctx context.Context, options modulepublish.Options) (*barnpublish.Result, error) {
	if s.publisher == nil {
		return nil, errors.New("module publisher is not configured")
	}

	return s.publisher.Prepare(ctx, options)
}
