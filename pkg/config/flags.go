package config

import "strings"

const (
	LoggerLevel  = "log-level"
	LoggerOutput = "log-output"
	LoggerFile   = "log-file"

	ExecRuntime             = "runtime"
	ExecRuntimeFSRoot       = "runtime-fs-root"
	ExecKeepCookies         = "browser-cookies"
	ExecWithBrowser         = "browser-open"
	ExecBrowserAddress      = "browser-address"
	ExecWithBrowserHeadless = "browser-headless"
	ExecProxy               = "proxy"
	ExecUserAgent           = "user-agent"

	PolicyHTTPAllowedSchemes        = "policy-http-allowed-schemes"
	PolicyHTTPAllowedMethods        = "policy-http-allowed-methods"
	PolicyHTTPAllowedHosts          = "policy-http-allowed-hosts"
	PolicyHTTPBlockedHosts          = "policy-http-blocked-hosts"
	PolicyHTTPAllowLocalhost        = "policy-http-allow-localhost"
	PolicyHTTPAllowPrivateNetworks  = "policy-http-allow-private-networks"
	PolicyHTTPAllowLinkLocal        = "policy-http-allow-link-local"
	PolicyHTTPDefaultHeaders        = "policy-http-default-headers"
	PolicyHTTPBlockedRequestHeaders = "policy-http-blocked-request-headers"
	PolicyHTTPTimeout               = "policy-http-timeout"
	PolicyHTTPNoTimeout             = "policy-http-no-timeout"
	PolicyHTTPMaxRequestSize        = "policy-http-max-request-size"
	PolicyHTTPUnlimitedRequestSize  = "policy-http-unlimited-request-size"
	PolicyHTTPMaxResponseSize       = "policy-http-max-response-size"
	PolicyHTTPUnlimitedResponseSize = "policy-http-unlimited-response-size"
	PolicyHTTPMaxResponseHeaderSize = "policy-http-max-response-header-size"
	PolicyHTTPFollowRedirects       = "policy-http-follow-redirects"
	PolicyHTTPMaxRedirects          = "policy-http-max-redirects"

	BrowserPort     = "port"
	BrowserDetach   = "detach"
	BrowserHeadless = "headless"
	BrowserUserDir  = "user-dir"
)

var Flags = []string{
	LoggerLevel,
	LoggerOutput,
	LoggerFile,
	ExecRuntime,
	ExecRuntimeFSRoot,
	ExecKeepCookies,
	ExecBrowserAddress,
	ExecWithBrowser,
	ExecWithBrowserHeadless,
	ExecProxy,
	ExecUserAgent,
	PolicyHTTPAllowedSchemes,
	PolicyHTTPAllowedMethods,
	PolicyHTTPAllowedHosts,
	PolicyHTTPBlockedHosts,
	PolicyHTTPAllowLocalhost,
	PolicyHTTPAllowPrivateNetworks,
	PolicyHTTPAllowLinkLocal,
	PolicyHTTPDefaultHeaders,
	PolicyHTTPBlockedRequestHeaders,
	PolicyHTTPTimeout,
	PolicyHTTPNoTimeout,
	PolicyHTTPMaxRequestSize,
	PolicyHTTPUnlimitedRequestSize,
	PolicyHTTPMaxResponseSize,
	PolicyHTTPUnlimitedResponseSize,
	PolicyHTTPMaxResponseHeaderSize,
	PolicyHTTPFollowRedirects,
	PolicyHTTPMaxRedirects,
}
var FlagsStr = strings.Join(Flags, `"|"`)

func isSupportedFlag(name string) bool {
	for _, f := range Flags {
		if f == name {
			return true
		}
	}

	return false
}
