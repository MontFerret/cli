package execution

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

// OptionsFromCommand overlays explicit command policies onto the store-backed runtime options.
func OptionsFromCommand(cmd *cobra.Command, store *config.Store) (cliruntime.Options, error) {
	opts := store.GetRuntimeOptions()

	httpPolicy, err := HTTPPolicyOptionsFromCommand(cmd)
	if err != nil {
		return cliruntime.Options{}, err
	}
	opts.HTTPPolicy = httpPolicy

	fsPolicy, err := FSPolicyFromCommand(cmd)
	if err != nil {
		return cliruntime.Options{}, err
	}
	opts.FSPolicy = fsPolicy

	return opts, nil
}
