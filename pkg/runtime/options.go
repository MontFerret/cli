package runtime

import (
	"fmt"

	"github.com/MontFerret/cli/v2/pkg/logger"
	"github.com/MontFerret/contrib/modules/web/html/drivers"
	"github.com/MontFerret/contrib/modules/web/html/drivers/cdp"
	"github.com/MontFerret/contrib/modules/web/html/drivers/memory"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
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
	Logger              logger.Options
	// FSPolicy configures filesystem access for the builtin runtime only.
	FSPolicy *FileSystemPolicy
	// HTTPPolicy configures outbound HTTP for the builtin runtime only.
	HTTPPolicy []ferrethttp.PolicyOption
}

// FileSystemPolicy configures the sandboxed filesystem used by the builtin runtime.
type FileSystemPolicy struct {
	Root     string
	ReadOnly bool
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
		Logger:              logger.NewDefaultOptions(),
	}
}

func ValidateOptions(opts Options) error {
	opts = NormalizeOptions(opts)

	if err := opts.Logger.Validate(); err != nil {
		return err
	}

	if len(opts.HTTPPolicy) > 0 {
		if !IsBuiltinType(opts.Type) {
			return ErrHTTPPolicyRequiresBuiltinRuntime
		}

		if _, err := ferrethttp.NewPolicy(opts.HTTPPolicy...); err != nil {
			return fmt.Errorf("HTTP policy: %w", err)
		}
	}

	if opts.FSPolicy != nil && !IsBuiltinType(opts.Type) {
		return ErrFSPolicyRequiresBuiltinRuntime
	}

	return nil
}

func NormalizeOptions(opts Options) Options {
	opts.Logger = logger.NormalizeOptions(opts.Logger)

	return opts
}

func (opts *Options) ToInMemory() []memory.Option {
	result := make([]memory.Option, 0, 4)

	if opts.Proxy != "" {
		result = append(result, memory.WithProxy(opts.Proxy))
	}

	if opts.UserAgent != "" {
		result = append(result, memory.WithUserAgent(opts.UserAgent))
	}

	if opts.Headers != nil {
		result = append(result, memory.WithHeaders(opts.Headers))
	}

	if opts.Cookies != nil {
		cookies := make([]drivers.HTTPCookie, 0, len(opts.Cookies.Data))

		for _, cookie := range opts.Cookies.Data {
			cookies = append(cookies, cookie)
		}

		result = append(result, memory.WithCookies(cookies))
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
		result = append(result, cdp.WithHeaders(opts.Headers.Data))
	}

	if opts.Cookies != nil {
		cookies := make([]drivers.HTTPCookie, 0, len(opts.Cookies.Data))

		for _, cookie := range opts.Cookies.Data {
			cookies = append(cookies, cookie)
		}

		result = append(result, cdp.WithCookies(cookies))
	}

	if opts.KeepCookies {
		result = append(result, cdp.WithKeepCookies())
	}

	return result
}
