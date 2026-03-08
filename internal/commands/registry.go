package commands

import (
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage container registry credentials",
	}

	cmd.AddCommand(newRegistryListCmd())
	cmd.AddCommand(newRegistryCreateCmd())
	cmd.AddCommand(newRegistryUpdateCmd())
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

			query := `query { myself { containerRegistryCreds { id name } } }`

			var result struct {
				Myself struct {
					ContainerRegistryCreds []api.Registry `json:"containerRegistryCreds"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.ContainerRegistryCreds)
		},
	}
}

func newRegistryCreateCmd() *cobra.Command {
	var (
		name     string
		username string
		password string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Add a container registry credential",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || username == "" || password == "" {
				exitError("validation_error", "--name, --username, and --password are required")
			}

			input := map[string]any{
				"name":     name,
				"username": username,
				"password": password,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "registry_create",
					"input": map[string]any{
						"name":     name,
						"username": username,
						"password": "***",
					},
				})
			}

			client := getClient()

			query := `mutation($input: SaveRegistryAuthInput!) {
				saveRegistryAuth(input: $input) { id name }
			}`

			vars := map[string]any{"input": input}

			var result struct {
				SaveRegistryAuth api.Registry `json:"saveRegistryAuth"`
			}
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.SaveRegistryAuth)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Registry name (required)")
	cmd.Flags().StringVar(&username, "username", "", "Registry username (required)")
	cmd.Flags().StringVar(&password, "password", "", "Registry password (required)")

	return cmd
}

func newRegistryUpdateCmd() *cobra.Command {
	var (
		username string
		password string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a container registry credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || password == "" {
				exitError("validation_error", "--username and --password are required")
			}

			input := map[string]any{
				"id":       args[0],
				"username": username,
				"password": password,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "registry_update",
					"input": map[string]any{
						"id":       args[0],
						"username": username,
						"password": "***",
					},
				})
			}

			client := getClient()

			query := `mutation($input: UpdateRegistryAuthInput!) {
				updateRegistryAuth(input: $input) { id name }
			}`

			vars := map[string]any{"input": input}

			var result struct {
				UpdateRegistryAuth api.Registry `json:"updateRegistryAuth"`
			}
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.UpdateRegistryAuth)
		},
	}

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

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run":     true,
					"action":      "registry_delete",
					"registry_id": args[0],
				})
			}

			client := getClient()

			query := `mutation($id: String!) {
				deleteRegistryAuth(registryAuthId: $id)
			}`

			vars := map[string]any{"id": args[0]}

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
