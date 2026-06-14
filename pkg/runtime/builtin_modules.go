package runtime

import (
	"fmt"

	"github.com/MontFerret/contrib/modules/csv"
	"github.com/MontFerret/contrib/modules/db/sqlite"
	"github.com/MontFerret/contrib/modules/security/jwt"
	"github.com/MontFerret/contrib/modules/toml"
	"github.com/MontFerret/contrib/modules/web/article"
	"github.com/MontFerret/contrib/modules/web/robots"
	"github.com/MontFerret/contrib/modules/web/sitemap"
	"github.com/MontFerret/contrib/modules/xml"
	"github.com/MontFerret/contrib/modules/yaml"
	"github.com/MontFerret/ferret/v2/pkg/module"

	"github.com/MontFerret/contrib/modules/web/html"
	"github.com/MontFerret/contrib/modules/web/html/drivers/cdp"
	"github.com/MontFerret/contrib/modules/web/html/drivers/memory"
)

func newModules(opts Options) ([]module.Module, error) {
	webmods, err := newWebModules(opts)

	if err != nil {
		return nil, err
	}

	datamods, err := newDataModules()

	if err != nil {
		return nil, err
	}

	return append(webmods, datamods...), nil
}

func newWebModules(opts Options) ([]module.Module, error) {
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
		jwt.New(),
		sqlite.New(),
	}, nil
}

func newDataModules() ([]module.Module, error) {
	return []module.Module{
		csv.New(),
		toml.New(),
		xml.New(),
		yaml.New(),
	}, nil
}
