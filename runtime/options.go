package runtime

import (
	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/drivers/http"
)

type Options struct {
	Type                string
	Proxy               string
	UserAgent           string
	Headers             *drivers.HTTPHeaders
	Cookies             *drivers.HTTPCookies
	KeepCookies         bool
	BrowserAddress      string
	WithBrowser         bool
	WithHeadlessBrowser bool
}

func NewDefaultOptions() Options {
	return Options{
		Type:                DefaultRuntime,
		BrowserAddress:      cdp.DefaultAddress,
		Proxy:               "",
		UserAgent:           "",
		Headers:             nil,
		Cookies:             nil,
		KeepCookies:         false,
		WithBrowser:         false,
		WithHeadlessBrowser: false,
	}
}

func (opts *Options) ToInMemory() []http.Option {
	result := make([]http.Option, 0, 4)

	if opts.Proxy != "" {
		result = append(result, http.WithProxy(opts.Proxy))
	}

	if opts.UserAgent != "" {
		result = append(result, http.WithUserAgent(opts.UserAgent))
	}

	if opts.Headers != nil {
		result = append(result, http.WithHeaders(opts.Headers))
	}

	if opts.Cookies != nil {
		result = append(result, http.WithCookies(opts.Cookies.Values()))
	}

	return result
}

func (opts *Options) ToCDP() []cdp.Option {
	result := make([]cdp.Option, 0, 6)

	if opts.BrowserAddress != "" {
		result = append(result, cdp.WithAddress(opts.BrowserAddress))
	}

	if opts.Proxy != "" {
		result = append(result, cdp.WithProxy(opts.Proxy))
	}

	if opts.UserAgent != "" {
		result = append(result, cdp.WithUserAgent(opts.UserAgent))
	}

	if opts.Headers != nil {
		result = append(result, cdp.WithHeaders(opts.Headers))
	}

	if opts.Cookies != nil {
		result = append(result, cdp.WithCookies(opts.Cookies.Values()))
	}

	if opts.KeepCookies {
		result = append(result, cdp.WithKeepCookies())
	}

	return result
}
