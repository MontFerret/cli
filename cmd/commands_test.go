package cmd

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
)

func TestCommandFacadePreservesCommandMetadata(t *testing.T) {
	store := new(config.Store)
	tests := []struct {
		name        string
		use         string
		aliases     []string
		subcommands []string
	}{
		{name: "browser", use: "browser", subcommands: []string{"close", "open"}},
		{name: "build", use: "build [files...]"},
		{name: "check", use: "check [files...]"},
		{name: "config", use: "config", subcommands: []string{"get", "list", "set", "unset"}},
		{name: "debug", use: "debug <script.fql>"},
		{name: "format", use: "fmt [files...]"},
		{name: "inspect", use: "inspect [script]"},
		{name: "migrate", use: "migrate", subcommands: []string{"check", "run"}},
		{name: "mod", use: "mod", subcommands: []string{"info", "init", "install", "publish", "search"}},
		{name: "repl", use: "repl"},
		{name: "run", use: "run [script]", aliases: []string{"exec"}},
		{name: "update", use: "update", subcommands: []string{"self"}},
		{name: "version", use: "version"},
	}
	commands := map[string]commandMetadata{
		"browser": commandMetadataFrom(BrowserCommand(store)),
		"build":   commandMetadataFrom(BuildCommand(store)),
		"check":   commandMetadataFrom(CheckCommand(store)),
		"config":  commandMetadataFrom(ConfigCommand(store)),
		"debug":   commandMetadataFrom(DebugCommand(store)),
		"format":  commandMetadataFrom(FormatCommand(store)),
		"inspect": commandMetadataFrom(InspectCommand(store)),
		"migrate": commandMetadataFrom(MigrateCommand(store, facadeMigrationService{})),
		"mod":     commandMetadataFrom(ModCommand(store, new(facadeModuleService))),
		"repl":    commandMetadataFrom(ReplCommand(store)),
		"run":     commandMetadataFrom(RunCommand(store)),
		"update":  commandMetadataFrom(SelfUpdateCommand(store)),
		"version": commandMetadataFrom(VersionCommand(store)),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commands[tt.name]
			if got.Use != tt.use {
				t.Fatalf("expected use %q, got %q", tt.use, got.Use)
			}
			if !slices.Equal(got.Aliases, tt.aliases) {
				t.Fatalf("expected aliases %#v, got %#v", tt.aliases, got.Aliases)
			}
			if !slices.Equal(got.Subcommands, tt.subcommands) {
				t.Fatalf("expected subcommands %#v, got %#v", tt.subcommands, got.Subcommands)
			}
		})
	}
}

type commandMetadata struct {
	Use         string
	Aliases     []string
	Subcommands []string
}

func commandMetadataFrom(command *cobra.Command) commandMetadata {
	subcommands := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		subcommands = append(subcommands, child.Name())
	}
	slices.Sort(subcommands)

	return commandMetadata{Use: command.Use, Aliases: command.Aliases, Subcommands: subcommands}
}
