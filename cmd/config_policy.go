package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

var policyConfigKeys = []string{
	config.PolicyFSRoot,
	config.PolicyFSReadOnly,
	config.PolicyHTTPAllowedSchemes,
	config.PolicyHTTPAllowedMethods,
	config.PolicyHTTPAllowedHosts,
	config.PolicyHTTPBlockedHosts,
	config.PolicyHTTPAllowLocalhost,
	config.PolicyHTTPAllowPrivateNetworks,
	config.PolicyHTTPAllowLinkLocal,
	config.PolicyHTTPDefaultHeaders,
	config.PolicyHTTPBlockedRequestHeaders,
	config.PolicyHTTPTimeout,
	config.PolicyHTTPNoTimeout,
	config.PolicyHTTPMaxRequestSize,
	config.PolicyHTTPUnlimitedRequestSize,
	config.PolicyHTTPMaxResponseSize,
	config.PolicyHTTPUnlimitedResponseSize,
	config.PolicyHTTPMaxResponseHeaderSize,
	config.PolicyHTTPFollowRedirects,
	config.PolicyHTTPMaxRedirects,
}

func validatePolicyConfigSet(store *config.Store, key, value string) error {
	if !isPolicyConfigKey(key) {
		return nil
	}

	command := &cobra.Command{Use: "config-policy-validation"}
	addFSPolicyFlags(command)
	addHTTPPolicyFlags(command)

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

	fsPolicy, err := fsPolicyFromCommand(command)
	if err != nil {
		return fmt.Errorf("invalid policy configuration for %q: %w", key, err)
	}

	httpPolicy, err := httpPolicyOptionsFromCommand(command)
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

func persistedPolicyValue(store *config.Store, policyKey, candidateKey, candidateValue string) (any, error) {
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
