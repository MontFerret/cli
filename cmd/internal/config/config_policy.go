package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	cliconfig "github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

var policyConfigKeys = []string{
	cliconfig.PolicyFSRoot,
	cliconfig.PolicyFSReadOnly,
	cliconfig.PolicyHTTPAllowedSchemes,
	cliconfig.PolicyHTTPAllowedMethods,
	cliconfig.PolicyHTTPAllowedHosts,
	cliconfig.PolicyHTTPBlockedHosts,
	cliconfig.PolicyHTTPAllowLocalhost,
	cliconfig.PolicyHTTPAllowPrivateNetworks,
	cliconfig.PolicyHTTPAllowLinkLocal,
	cliconfig.PolicyHTTPDefaultHeaders,
	cliconfig.PolicyHTTPBlockedRequestHeaders,
	cliconfig.PolicyHTTPTimeout,
	cliconfig.PolicyHTTPNoTimeout,
	cliconfig.PolicyHTTPMaxRequestSize,
	cliconfig.PolicyHTTPUnlimitedRequestSize,
	cliconfig.PolicyHTTPMaxResponseSize,
	cliconfig.PolicyHTTPUnlimitedResponseSize,
	cliconfig.PolicyHTTPMaxResponseHeaderSize,
	cliconfig.PolicyHTTPFollowRedirects,
	cliconfig.PolicyHTTPMaxRedirects,
}

func validatePolicyConfigSet(store *cliconfig.Store, key, value string) error {
	if !isPolicyConfigKey(key) {
		return nil
	}

	command := &cobra.Command{Use: "config-policy-validation"}
	execution.AddFSPolicyFlags(command)
	execution.AddHTTPPolicyFlags(command)

	for _, policyKey := range policyConfigKeys {
		policyValue, err := persistedPolicyValue(store, policyKey, key, value)
		if err != nil {
			return err
		}
		if policyValue == nil {
			continue
		}

		if err := command.Flags().Set(policyKey, fmt.Sprint(policyValue)); err != nil {
			return fmt.Errorf("invalid policy configuration for %q: %w", policyKey, err)
		}
	}

	fsPolicy, err := execution.FSPolicyFromCommand(command)
	if err != nil {
		return fmt.Errorf("invalid policy configuration for %q: %w", key, err)
	}

	httpPolicy, err := execution.HTTPPolicyOptionsFromCommand(command)
	if err != nil {
		return fmt.Errorf("invalid policy configuration for %q: %w", key, err)
	}

	opts := cliruntime.NewDefaultOptions()
	opts.FSPolicy = fsPolicy
	opts.HTTPPolicy = httpPolicy

	if err := cliruntime.ValidateOptions(opts); err != nil {
		return fmt.Errorf("invalid policy configuration for %q: %w", key, err)
	}

	return nil
}

func persistedPolicyValue(store *cliconfig.Store, policyKey, candidateKey, candidateValue string) (any, error) {
	if policyKey == candidateKey {
		return candidateValue, nil
	}

	return store.Get(policyKey)
}

func isPolicyConfigKey(key string) bool {
	for _, policyKey := range policyConfigKeys {
		if policyKey == key {
			return true
		}
	}

	return false
}
