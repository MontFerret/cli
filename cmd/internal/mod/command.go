package mod

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/config"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
)

const (
	moduleGoModuleFlag  = "go-module"
	moduleDirFlag       = "dir"
	moduleNamespaceFlag = "namespace"
	moduleTagFlag       = "tag"
	moduleDryRunFlag    = "dry-run"
	modulePrintFlag     = "print"
	moduleYesFlag       = "yes"
)

// New creates the module discovery, scaffolding, and publication command group.
func New(store *config.Store, service Service) *cobra.Command {
	return newCommand(store, service, defaultTerminal, newReadlinePrompt)
}

func newCommand(store *config.Store, service Service, terminal terminalDetector, prompts promptFactory) *cobra.Command {
	command := &cobra.Command{
		Use:   "mod",
		Short: "Discover, install, initialize, and publish Ferret modules",
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
		moduleInstallCommand(service, terminal, prompts),
		moduleInitCommand(service, terminal, prompts),
		modulePublishCommand(service),
	)

	return command
}

func moduleInstallCommand(service Service, terminal terminalDetector, prompts promptFactory) *cobra.Command {
	command := &cobra.Command{
		Use:   "install <module>[@version]",
		Short: "Install a registered module into the current Go application",
		Long: "Install a registered module into the current Go application.\n\n" +
			"The command updates go.mod, go.sum, and one unambiguous ferret.New(...) " +
			"composition. Module code is compiled into the application and executes " +
			"with the application's process permissions. It does not modify the Ferret CLI runtime. " +
			"When Ferret v2 or a safe composition helper is missing, interactive use shows the complete " +
			"project setup before proceeding; use --yes for non-interactive approval.",
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			yes, err := command.Flags().GetBool(moduleYesFlag)
			if err != nil {
				return err
			}

			fmt.Fprintf(command.OutOrStdout(), "Resolving %s...\n", args[0])

			result, proceed, err := installWithApproval(
				command.Context(),
				service,
				install.Options{
					Reference:                  args[0],
					Directory:                  ".",
					InstallMissingDependencies: yes,
					ScaffoldMissingComposition: yes,
				},
				terminal,
				prompts,
				command.InOrStdin(),
				command.ErrOrStderr(),
			)
			if err != nil || !proceed {
				return err
			}

			renderModuleInstall(command.OutOrStdout(), result)

			return nil
		},
	}

	command.Flags().BoolP(moduleYesFlag, "y", false, "Set up safe missing project prerequisites without prompting")

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

func moduleInitCommand(service Service, terminal terminalDetector, prompts promptFactory) *cobra.Command {
	command := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new Ferret module project",
		Long: "Initialize a new Ferret module project.\n\n" +
			"When run interactively, omitted values are explained and prompted with editable defaults. " +
			"For non-interactive use, provide the module name and --go-module.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			input, err := readInitInput(command, args)
			if err != nil {
				return err
			}

			options, proceed, err := resolveInitOptions(
				command.Context(),
				input,
				terminal,
				prompts,
				command.InOrStdin(),
				command.ErrOrStderr(),
			)
			if err != nil || !proceed {
				return err
			}

			result, err := service.Create(command.Context(), options)
			if err != nil {
				return err
			}

			renderModuleInit(command.OutOrStdout(), result)

			return nil
		},
	}

	command.Flags().String(moduleGoModuleFlag, "", "Go module import path (prompted when omitted interactively)")
	command.Flags().String(moduleDirFlag, "", "Destination directory (defaults to the module name leaf)")
	command.Flags().String(moduleNamespaceFlag, "", "Runtime namespace (defaults to the module name leaf)")

	return command
}

func modulePublishCommand(service Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "publish",
		Short: "Publish the current module release to the Ferret Registry",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(command *cobra.Command, _ []string) error {
			directory, err := command.Flags().GetString(moduleDirFlag)
			if err != nil {
				return err
			}

			if directory == "" {
				directory = "."
			}

			tag, err := command.Flags().GetString(moduleTagFlag)
			if err != nil {
				return err
			}

			dryRun, err := command.Flags().GetBool(moduleDryRunFlag)
			if err != nil {
				return err
			}

			printRecords, err := command.Flags().GetBool(modulePrintFlag)
			if err != nil {
				return err
			}

			if dryRun && printRecords {
				return fmt.Errorf("--dry-run and --print cannot be used together")
			}

			mode := modulepublish.ModeSubmit
			if dryRun {
				mode = modulepublish.ModeDryRun
			} else if printRecords {
				mode = modulepublish.ModePrint
			}

			publication, publishErr := service.Publish(
				command.Context(),
				modulepublish.Options{Directory: directory, Tag: tag, Mode: mode},
			)

			if publication != nil {
				if printRecords {
					if err := renderModulePublicationJSON(command.OutOrStdout(), publication); err != nil {
						return err
					}
				} else {
					renderModulePublication(command.OutOrStdout(), publication, mode)
				}
			}

			return publishErr
		},
	}

	command.Flags().String(moduleDirFlag, "", "Module directory (defaults to the current directory)")
	command.Flags().String(moduleTagFlag, "", "Release tag (defaults to v<version> or <source-path>/v<version>)")
	command.Flags().Bool(moduleDryRunFlag, false, "Validate and prepare the release without submitting to GitHub")
	command.Flags().Bool(modulePrintFlag, false, "Print deterministic Barn-relative records as JSON without submitting")

	return command
}
