package cmd

import (
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/pkg/config"
	cliruntime "github.com/MontFerret/cli/pkg/runtime"
)

func addRuntimeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(config.ExecRuntime, "r", cliruntime.DefaultRuntime, "Ferret runtime type (\"builtin\"|$url)")
	cmd.Flags().String(config.ExecProxy, "x", "Proxy server address")
	cmd.Flags().String(config.ExecUserAgent, "a", "User agent header")
	cmd.Flags().StringP(config.ExecBrowserAddress, "d", cliruntime.DefaultBrowser, "Browser debugger address")
	cmd.Flags().BoolP(config.ExecWithBrowser, "B", false, "Open browser for script execution")
	cmd.Flags().BoolP(config.ExecWithBrowserHeadless, "b", false, "Open browser for script execution in headless mode")
	cmd.Flags().BoolP(config.ExecKeepCookies, "c", false, "Keep cookies between queries")
}

func addParamFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayP(paramFlag, "p", []string{}, "Query bind parameter (--param=foo:\"bar\", --param=id:1)")
}

func addEvalFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("eval", "e", "", "Inline FQL expression to evaluate")
}
