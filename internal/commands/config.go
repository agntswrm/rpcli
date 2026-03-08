package commands

import (
	"github.com/agntswrm/rpcli/internal/config"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage rpcli configuration",
	}

	cmd.AddCommand(newConfigSetKeyCmd())
	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

func newConfigSetKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-key <api-key>",
		Short: "Store a Runpod API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetKey(args[0]); err != nil {
				exitError("config_error", err.Error())
			}
			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": "API key saved",
			})
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.Print(getFormat(), config.Show(apiKeyFlag))
		},
	}
}
