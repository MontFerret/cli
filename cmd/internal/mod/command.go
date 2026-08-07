package mod

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

const (
	moduleGoModuleFlag  = "go-module"
	moduleDirFlag       = "dir"
	moduleNamespaceFlag = "namespace"
	moduleTagFlag       = "tag"
)

// New creates the module discovery, scaffolding, and publication command group.
func New(store *config.Store, service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "mod",
		Short: "Discover, initialize, and publish Ferret modules",
		Args:  cobra.MaximumNArgs(0),
		PersistentPreRun: func(command *cobra.Command, _ []string) {
			store.BindFlags(command)
		},
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 0 {
				return command.Help()
			}

			return fmt.Errorf("unknown command %q", args[0])
		},
	}

	command.AddCommand(
		moduleSearchCommand(service),
		moduleInfoCommand(service),
		moduleInitCommand(service),
		modulePublishCommand(service),
	)

	return command
}

func moduleSearchCommand(service Service) *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search the Ferret module registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			query := ""
			if len(args) == 1 {
				query = args[0]
			}

			results, err := service.Search(command.Context(), query)
			if err != nil {
				return err
			}

			return renderModuleSearch(command.OutOrStdout(), results)
		},
	}
}

func moduleInfoCommand(service Service) *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show information about a registered Ferret module",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			info, err := service.Info(command.Context(), args[0])
			if err != nil {
				return err
			}

			renderModuleInfo(command.OutOrStdout(), info)

			return nil
		},
	}
}

func moduleInitCommand(service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "init <name>",
		Short: "Initialize a new Ferret module project",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			goModule, err := command.Flags().GetString(moduleGoModuleFlag)
			if err != nil {
				return err
			}

			if goModule == "" {
				return fmt.Errorf("--%s is required", moduleGoModuleFlag)
			}

			directory, err := command.Flags().GetString(moduleDirFlag)
			if err != nil {
				return err
			}

			namespace, err := command.Flags().GetString(moduleNamespaceFlag)
			if err != nil {
				return err
			}

			result, err := service.Create(command.Context(), modulelifecycle.CreateOptions{
				Name: args[0], GoModule: goModule, Directory: directory, Namespace: namespace,
			})
			if err != nil {
				return err
			}

			renderModuleInit(command.OutOrStdout(), result)

			return nil
		},
	}

	command.Flags().String(moduleGoModuleFlag, "", "Go module import path for the generated project")
	command.Flags().String(moduleDirFlag, "", "Destination directory (defaults to the module name leaf)")
	command.Flags().String(moduleNamespaceFlag, "", "Runtime namespace (defaults to the module name leaf)")

	return command
}

func modulePublishCommand(service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "publish",
		Short: "Prepare Barn registration records for the current module release",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(command *cobra.Command, _ []string) error {
			tag, err := command.Flags().GetString(moduleTagFlag)
			if err != nil {
				return err
			}

			publication, err := service.Publish(command.Context(), modulelifecycle.PublishOptions{Directory: ".", Tag: tag})
			if err != nil {
				return err
			}

			renderModulePublication(command.OutOrStdout(), publication)

			return nil
		},
	}

	command.Flags().String(moduleTagFlag, "", "Release tag (defaults to v<version> or <source-path>/v<version>)")

	return command
}
