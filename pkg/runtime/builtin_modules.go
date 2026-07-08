package runtime

import (
	"fmt"

	"github.com/MontFerret/contrib/modules/csv"
	"github.com/MontFerret/contrib/modules/db/postgres"
	"github.com/MontFerret/contrib/modules/db/sqlite"
	"github.com/MontFerret/contrib/modules/document/pdf"
	"github.com/MontFerret/contrib/modules/document/xlsx"
	"github.com/MontFerret/contrib/modules/net/rest"
	"github.com/MontFerret/contrib/modules/security/jwt"
	"github.com/MontFerret/contrib/modules/toml"
	"github.com/MontFerret/contrib/modules/web/article"
	"github.com/MontFerret/contrib/modules/web/html"
	"github.com/MontFerret/contrib/modules/web/html/drivers/cdp"
	"github.com/MontFerret/contrib/modules/web/html/drivers/memory"
	"github.com/MontFerret/contrib/modules/web/robots"
	"github.com/MontFerret/contrib/modules/web/sitemap"
	"github.com/MontFerret/contrib/modules/xml"
	"github.com/MontFerret/contrib/modules/yaml"
	"github.com/MontFerret/ferret/v2/pkg/module"
)

type namespaceInitializer func(opts Options) ([]module.Module, error)

func newModules(opts Options) ([]module.Module, error) {
	return initModules(
		opts,
		webMods,
		dataMods,
		dbMods,
		securityMods,
		networkMods,
		documentMods,
	)
}

func initModules(opts Options, initializers ...namespaceInitializer) ([]module.Module, error) {
	var merged []module.Module

	for _, r := range initializers {
		modules, err := r(opts)

		if err != nil {
			return nil, err
		}

		merged = append(merged, modules...)
	}

	return merged, nil
}

func webMods(opts Options) ([]module.Module, error) {
	htmlmod, err := html.New(
		html.WithDefaultDriver(memory.New(opts.ToInMemory()...)),
		html.WithDrivers(
			cdp.New(opts.ToCDP()...),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("initialize html module: %w", err)
	}

	return []module.Module{
		htmlmod,
		sitemap.New(),
		article.New(),
		robots.New(),
	}, nil
}

func dataMods(_ Options) ([]module.Module, error) {
	return []module.Module{
		csv.New(),
		toml.New(),
		xml.New(),
		yaml.New(),
	}, nil
}

func dbMods(_ Options) ([]module.Module, error) {
	return []module.Module{
		postgres.New(),
		sqlite.New(),
	}, nil
}

func securityMods(_ Options) ([]module.Module, error) {
	return []module.Module{
		jwt.New(),
	}, nil
}

func networkMods(_ Options) ([]module.Module, error) {
	return []module.Module{
		rest.New(),
	}, nil
}

func documentMods(_ Options) ([]module.Module, error) {
	return []module.Module{
		pdf.New(),
		xlsx.New(),
	}, nil
}
