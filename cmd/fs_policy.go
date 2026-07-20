package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func addFSPolicyFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.String(config.PolicyFSRoot, "", "Filesystem root directory for the builtin runtime")
	flags.Bool(config.PolicyFSReadOnly, false, "Make the builtin runtime filesystem read-only")
}

func fsPolicyFromCommand(cmd *cobra.Command) (*cliruntime.FileSystemPolicy, error) {
	if cmd == nil {
		return nil, nil
	}

	flags := cmd.Flags()
	rootSet := flags.Changed(config.PolicyFSRoot)
	readOnlySet := flags.Changed(config.PolicyFSReadOnly)
	if !rootSet && !readOnlySet {
		return nil, nil
	}

	policy := &cliruntime.FileSystemPolicy{}

	if rootSet {
		root, err := flags.GetString(config.PolicyFSRoot)
		if err != nil {
			return nil, err
		}

		policy.Root = strings.TrimSpace(root)
		if policy.Root == "" {
			return nil, fmt.Errorf("--%s cannot be empty", config.PolicyFSRoot)
		}
	}

	if readOnlySet {
		readOnly, err := flags.GetBool(config.PolicyFSReadOnly)
		if err != nil {
			return nil, err
		}

		policy.ReadOnly = readOnly
	}

	return policy, nil
}
