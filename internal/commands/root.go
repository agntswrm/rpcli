package commands

import (
	"os"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/config"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

var (
	apiKeyFlag  string
	outputFlag  string
	dryRunFlag  bool
	yesFlag     bool

	// Version is set at build time.
	Version = "dev"
)

func getFormat() output.Format {
	switch outputFlag {
	case "table":
		return output.FormatTable
	case "yaml":
		return output.FormatYAML
	default:
		return output.FormatJSON
	}
}

func getClient() *api.Client {
	key := config.ResolveAPIKey(apiKeyFlag)
	return api.NewClient(key, Version)
}

func exitError(code, message string) {
	output.PrintError(getFormat(), code, message)
	os.Exit(1)
}

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "rpcli",
		Short:         "Agent-first CLI for Runpod infrastructure",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "Runpod API key (overrides env and config)")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "json", "Output format: json, table, yaml")
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without making changes")
	rootCmd.PersistentFlags().BoolVar(&yesFlag, "yes", false, "Skip confirmation for destructive operations")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newGPUCmd())
	rootCmd.AddCommand(newCPUCmd())
	rootCmd.AddCommand(newPodCmd())
	rootCmd.AddCommand(newEndpointCmd())
	rootCmd.AddCommand(newTemplateCmd())
	rootCmd.AddCommand(newVolumeCmd())
	rootCmd.AddCommand(newRegistryCmd())
	rootCmd.AddCommand(newSecretCmd())
	rootCmd.AddCommand(newBillingCmd())
	rootCmd.AddCommand(newSSHCmd())
	rootCmd.AddCommand(newDoctorCmd())

	return rootCmd
}
