package config

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/MontFerret/cli/browser"
	"github.com/MontFerret/cli/logger"
	"github.com/MontFerret/cli/runtime"
)

type (
	KV struct {
		Key   string
		Value interface{}
	}

	Store struct {
		appName   string
		version   string
		envPrefix string
		v         *viper.Viper
	}
)

func NewStore(appName, version string) (*Store, error) {
	v := viper.New()

	if err := ensureConfigFile(v, appName); err != nil {
		return nil, err
	}

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	envPrefix := strings.ToUpper(appName)

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	return &Store{appName, version, envPrefix, v}, nil
}

func (s *Store) AppName() string {
	return s.appName
}

func (s *Store) AppVersion() string {
	return s.version
}

// Bind the current command's flags to viper
func (s *Store) BindFlags(cmd *cobra.Command) {
	bindFlagsFor(s.v, cmd, s.envPrefix)
}

func (s *Store) GetLoggerOptions() logger.Options {
	opts := logger.NewDefaultOptions()

	if s.v.IsSet(LoggerLevel) {
		opts.Level = logger.ToLevel(s.v.GetString(LoggerLevel))
	}

	return opts
}

func (s *Store) GetRuntimeOptions() runtime.Options {
	opts := runtime.NewDefaultOptions()

	if s.v.IsSet(ExecRuntime) {
		opts.Type = s.v.GetString(ExecRuntime)
	}

	if s.v.IsSet(ExecBrowserAddress) {
		opts.BrowserAddress = s.v.GetString(ExecBrowserAddress)
	}

	if s.v.IsSet(ExecKeepCookies) {
		opts.KeepCookies = s.v.GetBool(ExecKeepCookies)
	}

	if s.v.IsSet(ExecWithBrowserHeadless) {
		opts.WithHeadlessBrowser = s.v.GetBool(ExecWithBrowserHeadless)
	}

	if s.v.IsSet(ExecWithBrowser) {
		opts.WithBrowser = s.v.GetBool(ExecWithBrowser)
	} else if opts.WithHeadlessBrowser {
		opts.WithBrowser = true
	}

	if s.v.IsSet(ExecProxy) {
		opts.Proxy = s.v.GetString(ExecProxy)
	}

	if s.v.IsSet(ExecUserAgent) {
		opts.UserAgent = s.v.GetString(ExecUserAgent)
	}

	return opts
}

func (s *Store) GetBrowserOptions() browser.Options {
	opts := browser.NewDefaultOptions()

	if s.v.IsSet(BrowserDetach) {
		opts.Detach = s.v.GetBool(BrowserDetach)
	}

	if s.v.IsSet(BrowserHeadless) {
		opts.Headless = s.v.GetBool(BrowserHeadless)
	}

	if s.v.IsSet(BrowserPort) {
		opts.Port = s.v.GetUint64(BrowserPort)
	}

	if s.v.IsSet(BrowserUserDir) {
		opts.UserDir = s.v.GetString(BrowserUserDir)
	}

	return opts
}

func (s *Store) Get(key string) (interface{}, error) {
	if !isSupportedFlag(key) {
		return nil, ErrInvalidFlag
	}

	return s.v.Get(key), nil
}

func (s *Store) Set(key, val string) error {
	if !isSupportedFlag(key) {
		return ErrInvalidFlag
	}

	s.v.Set(key, val)

	return s.v.WriteConfig()
}

func (s *Store) List() []KV {
	list := make([]KV, 0, len(Flags))

	for _, key := range Flags {
		list = append(list, KV{
			Key:   key,
			Value: s.v.Get(key),
		})
	}

	return list
}
