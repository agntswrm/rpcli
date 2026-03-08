package commands

import (
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets",
	}

	cmd.AddCommand(newSecretListCmd())
	cmd.AddCommand(newSecretCreateCmd())
	cmd.AddCommand(newSecretDeleteCmd())

	return cmd
}

func newSecretListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all secrets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query { myself { secrets { name createdAt } } }`

			var result struct {
				Myself struct {
					Secrets []api.Secret `json:"secrets"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.Secrets)
		},
	}
}

func newSecretCreateCmd() *cobra.Command {
	var (
		name  string
		value string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || value == "" {
				exitError("validation_error", "--name and --value are required")
			}

			input := map[string]any{
				"name":  name,
				"value": value,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "secret_create",
					"input": map[string]any{
						"name":  name,
						"value": "***",
					},
				})
			}

			client := getClient()

			query := `mutation($input: CreateSecretInput!) {
				createSecret(input: $input) { name createdAt }
			}`

			vars := map[string]any{"input": input}

			var result struct {
				CreateSecret api.Secret `json:"createSecret"`
			}
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.CreateSecret)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Secret name (required)")
	cmd.Flags().StringVar(&value, "value", "", "Secret value (required)")

	return cmd
}

func newSecretDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"name": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "secret_delete",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: DeleteSecretInput!) {
				deleteSecret(input: $input)
			}`

			vars := map[string]any{"input": input}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Secret %s deleted", args[0]),
			})
		},
	}
}
