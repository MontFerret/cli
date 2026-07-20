package cmd

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func runtimeOptionsFromCommand(cmd *cobra.Command, store *config.Store) (cliruntime.Options, error) {
	opts := store.GetRuntimeOptions()

	httpPolicy, err := httpPolicyOptionsFromCommand(cmd)
	if err != nil {
		return cliruntime.Options{}, err
	}
	opts.HTTPPolicy = httpPolicy

	fsPolicy, err := fsPolicyFromCommand(cmd)
	if err != nil {
		return cliruntime.Options{}, err
	}
	opts.FSPolicy = fsPolicy

	return opts, nil
}
