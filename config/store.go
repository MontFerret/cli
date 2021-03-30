package config

import (
	"github.com/MontFerret/cli/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"

	"github.com/MontFerret/cli/runtime"
)

const (
	LoggerLevel        = "log-level"
	RuntimeType        = "runtime"
	RuntimeCDPAddress  = "browser"
	RuntimeProxy       = "proxy"
	RuntimeUserAgent   = "user-agent"
	RuntimeKeepCookies = "browser-cookies"
)

type Store struct {
	appName   string
	version   string
	envPrefix string
	v         *viper.Viper
}

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

	if s.v.IsSet(RuntimeType) {
		opts.Type = s.v.GetString(RuntimeType)
	}

	if s.v.IsSet(RuntimeCDPAddress) {
		opts.CDPAddress = s.v.GetString(RuntimeCDPAddress)
	}

	if s.v.IsSet(RuntimeProxy) {
		opts.Proxy = s.v.GetString(RuntimeProxy)
	}

	if s.v.IsSet(RuntimeUserAgent) {
		opts.UserAgent = s.v.GetString(RuntimeUserAgent)
	}

	if s.v.IsSet(RuntimeKeepCookies) {
		opts.KeepCookies = s.v.GetBool(RuntimeKeepCookies)
	}

	return opts
}
