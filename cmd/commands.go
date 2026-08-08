package cmd

import (
	"github.com/spf13/cobra"

	browsercmd "github.com/MontFerret/cli/v2/cmd/internal/browser"
	buildcmd "github.com/MontFerret/cli/v2/cmd/internal/build"
	checkcmd "github.com/MontFerret/cli/v2/cmd/internal/check"
	configcmd "github.com/MontFerret/cli/v2/cmd/internal/config"
	debugcmd "github.com/MontFerret/cli/v2/cmd/internal/debug"
	formatcmd "github.com/MontFerret/cli/v2/cmd/internal/format"
	inspectcmd "github.com/MontFerret/cli/v2/cmd/internal/inspect"
	modcmd "github.com/MontFerret/cli/v2/cmd/internal/mod"
	replcmd "github.com/MontFerret/cli/v2/cmd/internal/repl"
	runcmd "github.com/MontFerret/cli/v2/cmd/internal/run"
	updatecmd "github.com/MontFerret/cli/v2/cmd/internal/update"
	versioncmd "github.com/MontFerret/cli/v2/cmd/internal/version"
	"github.com/MontFerret/cli/v2/pkg/config"
)

type moduleService = modcmd.Service

// BrowserCommand creates the browser management command group.
func BrowserCommand(store *config.Store) *cobra.Command {
	return browsercmd.New(store)
}

// BuildCommand creates the FQL build command.
func BuildCommand(store *config.Store) *cobra.Command {
	return buildcmd.New(store)
}

// CheckCommand creates the FQL validation command.
func CheckCommand(store *config.Store) *cobra.Command {
	return checkcmd.New(store)
}

// ConfigCommand creates the configuration management command group.
func ConfigCommand(store *config.Store) *cobra.Command {
	return configcmd.New(store)
}

// DebugCommand creates the interactive debugger command.
func DebugCommand(store *config.Store) *cobra.Command {
	return debugcmd.New(store)
}

// FormatCommand creates the FQL formatting command.
func FormatCommand(store *config.Store) *cobra.Command {
	return formatcmd.New(store)
}

// InspectCommand creates the bytecode inspection command.
func InspectCommand(store *config.Store) *cobra.Command {
	return inspectcmd.New(store)
}

// ModCommand creates the module discovery, installation, scaffolding, and publication command group.
func ModCommand(store *config.Store, service moduleService) *cobra.Command {
	return modcmd.New(store, service)
}

// ReplCommand creates the interactive FQL shell command.
func ReplCommand(store *config.Store) *cobra.Command {
	return replcmd.New(store)
}

// RunCommand creates the FQL execution command.
func RunCommand(store *config.Store) *cobra.Command {
	return runcmd.New(store)
}

// SelfUpdateCommand creates the CLI self-update command group.
func SelfUpdateCommand(store *config.Store) *cobra.Command {
	return updatecmd.New(store)
}

// VersionCommand creates the CLI and runtime version command.
func VersionCommand(store *config.Store) *cobra.Command {
	return versioncmd.New(store)
}
