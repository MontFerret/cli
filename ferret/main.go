package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	barnregistry "github.com/MontFerret/barn/pkg/registry"

	"github.com/MontFerret/cli/v2/cmd"
	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/logger"
	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	githubpublish "github.com/MontFerret/cli/v2/pkg/module/publish/github"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

const (
	name = "ferret"
)

var version string

func main() {
	store, err := config.NewStore(name, version)
	if err != nil {
		exit(err)
	}

	rootCmd := &cobra.Command{
		Use:              name,
		SilenceErrors:    true,
		SilenceUsage:     true,
		TraverseChildren: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return fmt.Errorf("unknown command %q", args[0])
		},
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringP(config.LoggerLevel, "l", zerolog.InfoLevel.String(), fmt.Sprintf("Set the logging level (%s)", logger.LevelsFmt()))
	rootCmd.PersistentFlags().String(config.LoggerOutput, logger.OutputStderr, fmt.Sprintf("Set the query execution log output (%s)", logger.OutputsFmt()))
	rootCmd.PersistentFlags().String(config.LoggerFile, "ferret.log", "Set the query execution log file path when --log-output=file")

	registryClient, err := barnregistry.NewClient()
	if err != nil {
		exit(err)
	}

	registrySubmitter, err := githubpublish.New()
	if err != nil {
		exit(err)
	}

	moduleService := modulelifecycle.NewService(
		discovery.New(registryClient),
		install.New(registryClient, nil),
		scaffold.New(nil),
		modulepublish.New(registryClient, registrySubmitter),
	)

	rootCmd.AddCommand(
		cmd.VersionCommand(store),
		cmd.ConfigCommand(store),
		cmd.RunCommand(store),
		cmd.DebugCommand(store),
		cmd.ReplCommand(store),
		cmd.FormatCommand(store),
		cmd.CheckCommand(store),
		cmd.BuildCommand(store),
		cmd.InspectCommand(store),
		cmd.BrowserCommand(store),
		cmd.SelfUpdateCommand(store),
		cmd.ModCommand(store, moduleService),
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			<-c
			cancel()
		}
	}()

	if err := rootCmd.ExecuteContext(config.With(ctx, store)); err != nil {
		exit(err)
	}
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}
