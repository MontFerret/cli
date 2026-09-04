package runtime

import (
	"github.com/MontFerret/contrib/modules/ai/llm"
	"github.com/MontFerret/contrib/modules/archive"
	"github.com/MontFerret/contrib/modules/csv"
	"github.com/MontFerret/contrib/modules/db/postgres"
	"github.com/MontFerret/contrib/modules/db/sqlite"
	"github.com/MontFerret/contrib/modules/document/pdf"
	"github.com/MontFerret/contrib/modules/document/xlsx"
	"github.com/MontFerret/contrib/modules/net/rest"
	"github.com/MontFerret/contrib/modules/security/jwt"
	"github.com/MontFerret/contrib/modules/security/oauth2"
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

type namespaceInitializer func(opts Options) []module.Module

func newModules(opts Options) ([]module.Module, error) {
	return initModules(
		opts,
		webMods,
		dataMods,
		dbMods,
		securityMods,
		networkMods,
		documentMods,
		aiMods,
		archiveMods,
	)
}

func initModules(opts Options, initializers ...namespaceInitializer) ([]module.Module, error) {
	var merged []module.Module

	for _, r := range initializers {
		merged = append(merged, r(opts)...)
	}

	return merged, nil
}

func webMods(opts Options) []module.Module {
	return []module.Module{
		html.New(
			html.WithDefaultDriver(memory.New(opts.ToInMemory()...)),
			html.WithDrivers(
				cdp.New(opts.ToCDP()...),
			),
		),
		sitemap.New(),
		article.New(),
		robots.New(),
	}
}

func dataMods(_ Options) []module.Module {
	return []module.Module{
		csv.New(),
		toml.New(),
		xml.New(),
		yaml.New(),
	}
}

func dbMods(_ Options) []module.Module {
	return []module.Module{
		postgres.New(),
		sqlite.New(),
	}
}

func securityMods(_ Options) []module.Module {
	return []module.Module{
		jwt.New(),
		oauth2.New(),
	}
}

func networkMods(_ Options) []module.Module {
	return []module.Module{
		rest.New(),
	}
}

func documentMods(_ Options) []module.Module {
	return []module.Module{
		pdf.New(),
		xlsx.New(),
	}
}

func aiMods(_ Options) []module.Module {
	return []module.Module{
		llm.New(),
	}
}

func archiveMods(_ Options) []module.Module {
	return []module.Module{
		archive.New(),
	}
}
