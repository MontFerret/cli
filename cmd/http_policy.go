package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
)

const (
	defaultHTTPPolicyTimeout               = 30 * time.Second
	defaultHTTPPolicyMaxRequestSize  int64 = 16 << 20
	defaultHTTPPolicyMaxResponseSize int64 = 16 << 20
	defaultHTTPPolicyMaxHeaderSize   int64 = 1 << 20
	defaultHTTPPolicyMaxRedirects          = 10
)

func addHTTPPolicyFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringSlice(config.PolicyHTTPAllowedSchemes, []string{"http", "https"}, "Allowed outbound HTTP URL schemes")
	flags.StringSlice(config.PolicyHTTPAllowedMethods, []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, "Allowed outbound HTTP methods")
	flags.StringSlice(config.PolicyHTTPAllowedHosts, nil, "Allowed outbound HTTP hosts (exact host or host:port)")
	flags.StringSlice(config.PolicyHTTPBlockedHosts, nil, "Blocked outbound HTTP hosts (exact host or host:port)")
	flags.Bool(config.PolicyHTTPAllowLocalhost, false, "Allow outbound HTTP access to localhost and loopback addresses")
	flags.Bool(config.PolicyHTTPAllowPrivateNetworks, false, "Allow outbound HTTP access to private network addresses")
	flags.Bool(config.PolicyHTTPAllowLinkLocal, false, "Allow outbound HTTP access to link-local addresses")
	flags.String(config.PolicyHTTPDefaultHeaders, "", "Default outbound HTTP headers as a JSON object")
	flags.StringSlice(config.PolicyHTTPBlockedRequestHeaders, nil, "Blocked outbound HTTP request header names")
	flags.Duration(config.PolicyHTTPTimeout, defaultHTTPPolicyTimeout, "Overall outbound HTTP timeout")
	flags.Bool(config.PolicyHTTPNoTimeout, false, "Disable the overall outbound HTTP timeout")
	flags.Int64(config.PolicyHTTPMaxRequestSize, defaultHTTPPolicyMaxRequestSize, "Maximum outbound HTTP request body size in bytes")
	flags.Bool(config.PolicyHTTPUnlimitedRequestSize, false, "Disable the outbound HTTP request body size limit")
	flags.Int64(config.PolicyHTTPMaxResponseSize, defaultHTTPPolicyMaxResponseSize, "Maximum outbound HTTP response body size in bytes")
	flags.Bool(config.PolicyHTTPUnlimitedResponseSize, false, "Disable the outbound HTTP response body size limit")
	flags.Int64(config.PolicyHTTPMaxResponseHeaderSize, defaultHTTPPolicyMaxHeaderSize, "Maximum outbound HTTP response header size in bytes")
	flags.Bool(config.PolicyHTTPFollowRedirects, true, "Follow outbound HTTP redirects")
	flags.Int(config.PolicyHTTPMaxRedirects, defaultHTTPPolicyMaxRedirects, "Maximum number of outbound HTTP redirects")
}

func runtimeOptionsFromCommand(cmd *cobra.Command, store *config.Store) (cliruntime.Options, error) {
	opts := store.GetRuntimeOptions()

	policy, err := httpPolicyOptionsFromCommand(cmd)
	if err != nil {
		return cliruntime.Options{}, err
	}

	opts.HTTPPolicy = policy

	return opts, nil
}

func httpPolicyOptionsFromCommand(cmd *cobra.Command) ([]ferrethttp.PolicyOption, error) {
	if cmd == nil {
		return nil, nil
	}

	flags := cmd.Flags()

	noTimeout, err := flags.GetBool(config.PolicyHTTPNoTimeout)
	if err != nil {
		return nil, err
	}
	if noTimeout && flags.Changed(config.PolicyHTTPTimeout) {
		return nil, fmt.Errorf("--%s cannot be combined with --%s", config.PolicyHTTPNoTimeout, config.PolicyHTTPTimeout)
	}

	unlimitedRequest, err := flags.GetBool(config.PolicyHTTPUnlimitedRequestSize)
	if err != nil {
		return nil, err
	}
	if unlimitedRequest && flags.Changed(config.PolicyHTTPMaxRequestSize) {
		return nil, fmt.Errorf("--%s cannot be combined with --%s", config.PolicyHTTPUnlimitedRequestSize, config.PolicyHTTPMaxRequestSize)
	}

	unlimitedResponse, err := flags.GetBool(config.PolicyHTTPUnlimitedResponseSize)
	if err != nil {
		return nil, err
	}
	if unlimitedResponse && flags.Changed(config.PolicyHTTPMaxResponseSize) {
		return nil, fmt.Errorf("--%s cannot be combined with --%s", config.PolicyHTTPUnlimitedResponseSize, config.PolicyHTTPMaxResponseSize)
	}

	var options []ferrethttp.PolicyOption

	if flags.Changed(config.PolicyHTTPAllowedSchemes) {
		value, err := flags.GetStringSlice(config.PolicyHTTPAllowedSchemes)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowedSchemes(value...))
	}

	if flags.Changed(config.PolicyHTTPAllowedMethods) {
		value, err := flags.GetStringSlice(config.PolicyHTTPAllowedMethods)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowedMethods(value...))
	}

	if flags.Changed(config.PolicyHTTPAllowedHosts) {
		value, err := flags.GetStringSlice(config.PolicyHTTPAllowedHosts)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowedHosts(value...))
	}

	if flags.Changed(config.PolicyHTTPBlockedHosts) {
		value, err := flags.GetStringSlice(config.PolicyHTTPBlockedHosts)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithBlockedHosts(value...))
	}

	if flags.Changed(config.PolicyHTTPAllowLocalhost) {
		value, err := flags.GetBool(config.PolicyHTTPAllowLocalhost)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowLocalhost(value))
	}

	if flags.Changed(config.PolicyHTTPAllowPrivateNetworks) {
		value, err := flags.GetBool(config.PolicyHTTPAllowPrivateNetworks)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowPrivateNetworks(value))
	}

	if flags.Changed(config.PolicyHTTPAllowLinkLocal) {
		value, err := flags.GetBool(config.PolicyHTTPAllowLinkLocal)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithAllowLinkLocal(value))
	}

	if flags.Changed(config.PolicyHTTPDefaultHeaders) {
		value, err := flags.GetString(config.PolicyHTTPDefaultHeaders)
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		if err := json.Unmarshal([]byte(value), &headers); err != nil {
			return nil, fmt.Errorf("invalid --%s: expected a JSON object of string values: %w", config.PolicyHTTPDefaultHeaders, err)
		}
		if headers == nil {
			return nil, fmt.Errorf("invalid --%s: expected a JSON object of string values", config.PolicyHTTPDefaultHeaders)
		}

		options = append(options, ferrethttp.WithDefaultHeaders(headers))
	}

	if flags.Changed(config.PolicyHTTPBlockedRequestHeaders) {
		value, err := flags.GetStringSlice(config.PolicyHTTPBlockedRequestHeaders)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithBlockedRequestHeaders(value...))
	}

	if noTimeout {
		options = append(options, ferrethttp.WithNoTimeout())
	} else if flags.Changed(config.PolicyHTTPTimeout) {
		value, err := flags.GetDuration(config.PolicyHTTPTimeout)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithTimeout(value))
	} else if flags.Changed(config.PolicyHTTPNoTimeout) {
		options = append(options, ferrethttp.WithTimeout(0))
	}

	if unlimitedRequest {
		options = append(options, ferrethttp.WithUnlimitedRequestSize())
	} else if flags.Changed(config.PolicyHTTPMaxRequestSize) {
		value, err := flags.GetInt64(config.PolicyHTTPMaxRequestSize)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithMaxRequestSize(value))
	} else if flags.Changed(config.PolicyHTTPUnlimitedRequestSize) {
		options = append(options, ferrethttp.WithMaxRequestSize(0))
	}

	if unlimitedResponse {
		options = append(options, ferrethttp.WithUnlimitedResponseSize())
	} else if flags.Changed(config.PolicyHTTPMaxResponseSize) {
		value, err := flags.GetInt64(config.PolicyHTTPMaxResponseSize)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithMaxResponseSize(value))
	} else if flags.Changed(config.PolicyHTTPUnlimitedResponseSize) {
		options = append(options, ferrethttp.WithMaxResponseSize(0))
	}

	if flags.Changed(config.PolicyHTTPMaxResponseHeaderSize) {
		value, err := flags.GetInt64(config.PolicyHTTPMaxResponseHeaderSize)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithMaxResponseHeaderSize(value))
	}

	if flags.Changed(config.PolicyHTTPFollowRedirects) {
		value, err := flags.GetBool(config.PolicyHTTPFollowRedirects)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithFollowRedirects(value))
	}

	if flags.Changed(config.PolicyHTTPMaxRedirects) {
		value, err := flags.GetInt(config.PolicyHTTPMaxRedirects)
		if err != nil {
			return nil, err
		}

		options = append(options, ferrethttp.WithMaxRedirects(value))
	}

	return options, nil
}
