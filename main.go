package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/cmd"
	"github.com/MontFerret/cli/config"
	"github.com/MontFerret/cli/logger"
)

const (
	name = "ferret"
)

var (
	version string
)

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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			store.BindFlags(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return fmt.Errorf("unknown command %q", args[0])
		},
	}

	rootCmd.PersistentFlags().StringP(config.LoggerLevel, "l", zerolog.InfoLevel.String(), fmt.Sprintf("Set the logging level (%s)", logger.LevelsFmt()))

	rootCmd.AddCommand(
		cmd.VersionCommand(store),
		cmd.ConfigCommand(store),
		cmd.ExecCommand(store),
		cmd.BrowserCommand(store),
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

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
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
