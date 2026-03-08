package commands

import (
	"encoding/json"
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const volumeFields = `id name size dataCenterId`

func newVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage network volumes",
	}

	cmd.AddCommand(newVolumeListCmd())
	cmd.AddCommand(newVolumeCreateCmd())
	cmd.AddCommand(newVolumeUpdateCmd())
	cmd.AddCommand(newVolumeDeleteCmd())

	return cmd
}

func newVolumeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all network volumes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { networkVolumes { %s } } }`, volumeFields)

			var result struct {
				Myself struct {
					NetworkVolumes []api.Volume `json:"networkVolumes"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.NetworkVolumes)
		},
	}
}

func newVolumeCreateCmd() *cobra.Command {
	var (
		name         string
		size         float64
		dataCenterID string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a network volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || size <= 0 || dataCenterID == "" {
				exitError("validation_error", "--name, --size, and --datacenter are required")
			}

			input := map[string]any{
				"name":         name,
				"size":         size,
				"dataCenterId": dataCenterID,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "volume_create",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: CreateNetworkVolumeInput!) {
				createNetworkVolume(input: $input) { %s }
			}`, volumeFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var vol api.Volume
			for _, v := range result {
				if err := json.Unmarshal(v, &vol); err == nil && vol.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), vol)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Volume name (required)")
	cmd.Flags().Float64Var(&size, "size", 0, "Volume size in GB (required)")
	cmd.Flags().StringVar(&dataCenterID, "datacenter", "", "Data center ID (required)")

	return cmd
}

func newVolumeUpdateCmd() *cobra.Command {
	var (
		name string
		size float64
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a network volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := map[string]any{
				"id": args[0],
			}
			if cmd.Flags().Changed("name") {
				input["name"] = name
			}
			if cmd.Flags().Changed("size") {
				input["size"] = size
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "volume_update",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: UpdateNetworkVolumeInput!) {
				updateNetworkVolume(input: $input) { %s }
			}`, volumeFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var vol api.Volume
			for _, v := range result {
				if err := json.Unmarshal(v, &vol); err == nil && vol.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), vol)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Volume name")
	cmd.Flags().Float64Var(&size, "size", 0, "Volume size in GB")

	return cmd
}

func newVolumeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a network volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"id": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "volume_delete",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: DeleteNetworkVolumeInput!) {
				deleteNetworkVolume(input: $input)
			}`

			vars := map[string]any{"input": input}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Volume %s deleted", args[0]),
			})
		},
	}
}
