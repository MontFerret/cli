package execution

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

// AddRuntimeFlags keeps the execution-policy surface identical across run, debug, and repl.
func AddRuntimeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(config.ExecRuntime, "r", cliruntime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().String(config.ExecProxy, "x", "Proxy server address")
	cmd.Flags().String(config.ExecUserAgent, "a", "User agent header")
	cmd.Flags().StringP(config.ExecBrowserAddress, "d", cliruntime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().BoolP(config.ExecWithBrowser, "B", false, "Open browser for script execution")
	cmd.Flags().BoolP(config.ExecWithBrowserHeadless, "b", false, "Open browser for script execution in headless mode")
	cmd.Flags().BoolP(config.ExecKeepCookies, "c", false, "Keep cookies between queries")
	AddFSPolicyFlags(cmd)
	AddHTTPPolicyFlags(cmd)
}

// AddParamFlags registers the repeatable JSON-aware runtime parameter flag.
func AddParamFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayP(ParamFlag, "p", []string{}, "Runtime parameter as name=value. Values parse as JSON when possible, otherwise strings. Examples: --param name=Steve, --param age=42, --param active=true, --param tags='[\"admin\",\"editor\"]', --param user='{\"name\":\"Ada\"}', --param code='\"123\"'")
}

// AddEvalFlag registers inline FQL input for commands that support it.
func AddEvalFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("eval", "e", "", "Inline FQL expression to evaluate")
}
