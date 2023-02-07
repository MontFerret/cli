package config

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func ensureConfigFile(v *viper.Viper, appName string) error {
	home, err := homedir.Dir()

	if err != nil {
		return err
	}

	projectDir := path.Join(home, "."+appName)

	_, err = os.Stat(projectDir)

	// first launch
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(projectDir, 0755); err != nil {
			return errors.Wrap(err, "create project directory")
		}
	}

	configFile := path.Join(projectDir, "config.yaml")

	_, err = os.Stat(configFile)

	if errors.Is(err, os.ErrNotExist) {
		if _, err := os.Create(configFile); err != nil {
			return errors.Wrap(err, "create project config file")
		}
	}

	// Set the base name of the config file, without the file extension.
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(projectDir)

	return nil
}

func bindFlags(v *viper.Viper, flags *pflag.FlagSet, envPrefix string) {
	flags.VisitAll(func(f *pflag.Flag) {
		v.BindPFlag(f.Name, f)

		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			flags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func bindFlagsFor(v *viper.Viper, cmd *cobra.Command, envPrefix string) {
	bindFlags(v, cmd.Flags(), envPrefix)
	bindFlags(v, cmd.PersistentFlags(), envPrefix)
}
