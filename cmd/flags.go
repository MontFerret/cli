package cmd

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func addRuntimeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(config.ExecRuntime, "r", cliruntime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().String(config.ExecProxy, "x", "Proxy server address")
	cmd.Flags().String(config.ExecUserAgent, "a", "User agent header")
	cmd.Flags().StringP(config.ExecBrowserAddress, "d", cliruntime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().BoolP(config.ExecWithBrowser, "B", false, "Open browser for script execution")
	cmd.Flags().BoolP(config.ExecWithBrowserHeadless, "b", false, "Open browser for script execution in headless mode")
	cmd.Flags().BoolP(config.ExecKeepCookies, "c", false, "Keep cookies between queries")
	addFSPolicyFlags(cmd)
	addHTTPPolicyFlags(cmd)
}

func addParamFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayP(paramFlag, "p", []string{}, "Runtime parameter as name=value. Values parse as JSON when possible, otherwise strings. Examples: --param name=Steve, --param age=42, --param active=true, --param tags='[\"admin\",\"editor\"]', --param user='{\"name\":\"Ada\"}', --param code='\"123\"'")
}

func addEvalFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("eval", "e", "", "Inline FQL expression to evaluate")
}
