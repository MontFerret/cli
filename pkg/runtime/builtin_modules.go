package runtime

import (
	"fmt"

	"github.com/MontFerret/contrib/modules/csv"
	"github.com/MontFerret/contrib/modules/toml"
	"github.com/MontFerret/contrib/modules/web/article"
	"github.com/MontFerret/contrib/modules/web/robots"
	"github.com/MontFerret/contrib/modules/web/sitemap"
	"github.com/MontFerret/contrib/modules/xml"
	"github.com/MontFerret/contrib/modules/yaml"

	"github.com/MontFerret/contrib/modules/web/html"
	"github.com/MontFerret/contrib/modules/web/html/drivers/cdp"
	"github.com/MontFerret/contrib/modules/web/html/drivers/http"
	"github.com/MontFerret/ferret/v2"
)

func newModules(opts Options) ([]ferret.Module, error) {
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

func newWebModules(opts Options) ([]ferret.Module, error) {
	htmlmod, err := html.New(
		html.WithDefaultDriver(http.NewDriver(opts.ToInMemory()...)),
		html.WithDrivers(
			cdp.NewDriver(opts.ToCDP()...),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("initialize html module: %w", err)
	}

	return []ferret.Module{
		htmlmod,
		sitemap.New(),
		article.New(),
		robots.New(),
	}, nil
}

func newDataModules() ([]ferret.Module, error) {
	return []ferret.Module{
		csv.New(),
		toml.New(),
		xml.New(),
		yaml.New(),
	}, nil
}
