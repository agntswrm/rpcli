package commands

import (
	"encoding/json"
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const registryFields = `id name url username`

func newRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage container registry credentials",
	}

	cmd.AddCommand(newRegistryListCmd())
	cmd.AddCommand(newRegistryCreateCmd())
	cmd.AddCommand(newRegistryDeleteCmd())

	return cmd
}

func newRegistryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List container registry credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { containerRegistries { %s } } }`, registryFields)

			var result struct {
				Myself struct {
					ContainerRegistries []api.Registry `json:"containerRegistries"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.ContainerRegistries)
		},
	}
}

func newRegistryCreateCmd() *cobra.Command {
	var (
		name     string
		url      string
		username string
		password string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Add a container registry credential",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || url == "" || username == "" || password == "" {
				exitError("validation_error", "--name, --url, --username, and --password are required")
			}

			input := map[string]any{
				"name":     name,
				"url":      url,
				"username": username,
				"password": password,
			}

			if dryRunFlag {
				// Mask password in dry-run output
				dryInput := map[string]any{
					"name":     name,
					"url":      url,
					"username": username,
					"password": "***",
				}
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "registry_create",
					"input":   dryInput,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: SaveRegistryInput!) {
				saveContainerRegistryAuth(input: $input) { %s }
			}`, registryFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var reg api.Registry
			for _, v := range result {
				if err := json.Unmarshal(v, &reg); err == nil && reg.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), reg)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Registry name (required)")
	cmd.Flags().StringVar(&url, "url", "", "Registry URL (required)")
	cmd.Flags().StringVar(&username, "username", "", "Registry username (required)")
	cmd.Flags().StringVar(&password, "password", "", "Registry password (required)")

	return cmd
}

func newRegistryDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a container registry credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"id": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "registry_delete",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: DeleteRegistryInput!) {
				deleteContainerRegistryAuth(input: $input)
			}`

			vars := map[string]any{"input": input}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Registry credential %s deleted", args[0]),
			})
		},
	}
}
