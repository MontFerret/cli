package config

import "strings"

const (
	LoggerLevel        = "log-level"
	RuntimeType        = "runtime"
	RuntimeCDPAddress  = "browser"
	RuntimeProxy       = "proxy"
	RuntimeUserAgent   = "user-agent"
	RuntimeKeepCookies = "browser-cookies"
	BrowserPort        = "port"
	BrowserDetach      = "detach"
	BrowserHeadless    = "headless"
	BrowserUserDir     = "user-dir"
)

var Flags = []string{LoggerLevel, RuntimeType, RuntimeCDPAddress, RuntimeProxy, RuntimeUserAgent, RuntimeKeepCookies}
var FlagsStr = strings.Join(Flags, `"|"`)

func isSupportedFlag(name string) bool {
	for _, f := range Flags {
		if f == name {
			return true
		}
	}

	return false
}
