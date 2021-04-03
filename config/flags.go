package config

import "strings"

const (
	LoggerLevel = "log-level"

	ExecRuntime             = "runtime"
	ExecKeepCookies         = "browser-cookies"
	ExecWithBrowser         = "browser-open"
	ExecBrowserAddress      = "browser-address"
	ExecWithBrowserHeadless = "browser-headless"
	ExecProxy               = "proxy"
	ExecUserAgent           = "user-agent"

	BrowserPort     = "port"
	BrowserDetach   = "detach"
	BrowserHeadless = "headless"
	BrowserUserDir  = "user-dir"
)

var Flags = []string{
	LoggerLevel,
	ExecRuntime,
	ExecKeepCookies,
	ExecBrowserAddress,
	ExecWithBrowser,
	ExecWithBrowserHeadless,
	ExecProxy,
	ExecUserAgent,
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
