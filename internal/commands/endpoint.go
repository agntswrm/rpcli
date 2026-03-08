package commands

import (
	"encoding/json"
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const endpointFields = `id name templateId gpuIds workersMin workersMax idleTimeout networkVolumeId`

func newEndpointCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "Manage serverless endpoints",
	}

	cmd.AddCommand(newEndpointListCmd())
	cmd.AddCommand(newEndpointGetCmd())
	cmd.AddCommand(newEndpointCreateCmd())
	cmd.AddCommand(newEndpointUpdateCmd())
	cmd.AddCommand(newEndpointDeleteCmd())
	cmd.AddCommand(newEndpointSwapTemplateCmd())

	return cmd
}

func newEndpointListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { endpoints { %s } } }`, endpointFields)

			var result struct {
				Myself struct {
					Endpoints []api.Endpoint `json:"endpoints"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.Endpoints)
		},
	}
}

func newEndpointGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get endpoint details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { endpoints { %s } } }`, endpointFields)

			var result struct {
				Myself struct {
					Endpoints []api.Endpoint `json:"endpoints"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			for _, ep := range result.Myself.Endpoints {
				if ep.ID == args[0] {
					return output.Print(getFormat(), ep)
				}
			}

			exitError("not_found", fmt.Sprintf("Endpoint %s not found", args[0]))
			return nil
		},
	}
}

func newEndpointCreateCmd() *cobra.Command {
	var (
		name        string
		templateID  string
		gpuIDs      string
		workersMin  int
		workersMax  int
		idleTimeout int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a serverless endpoint",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || templateID == "" || gpuIDs == "" {
				exitError("validation_error", "--name, --template-id, and --gpu-ids are required")
			}

			input := map[string]any{
				"name":       name,
				"templateId": templateID,
				"gpuIds":     gpuIDs,
				"workersMin": workersMin,
				"workersMax": workersMax,
			}
			if idleTimeout > 0 {
				input["idleTimeout"] = idleTimeout
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "endpoint_create",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: EndpointInput!) {
				saveEndpoint(input: $input) { %s }
			}`, endpointFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var ep api.Endpoint
			for _, v := range result {
				if err := json.Unmarshal(v, &ep); err == nil && ep.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), ep)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Endpoint name (required)")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Template ID (required)")
	cmd.Flags().StringVar(&gpuIDs, "gpu-ids", "", "GPU type IDs (required)")
	cmd.Flags().IntVar(&workersMin, "workers-min", 0, "Minimum number of workers")
	cmd.Flags().IntVar(&workersMax, "workers-max", 1, "Maximum number of workers")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")

	return cmd
}

func newEndpointUpdateCmd() *cobra.Command {
	var (
		name        string
		templateID  string
		gpuIDs      string
		workersMin  int
		workersMax  int
		idleTimeout int
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a serverless endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := map[string]any{
				"id": args[0],
			}
			if cmd.Flags().Changed("name") {
				input["name"] = name
			}
			if cmd.Flags().Changed("template-id") {
				input["templateId"] = templateID
			}
			if cmd.Flags().Changed("gpu-ids") {
				input["gpuIds"] = gpuIDs
			}
			if cmd.Flags().Changed("workers-min") {
				input["workersMin"] = workersMin
			}
			if cmd.Flags().Changed("workers-max") {
				input["workersMax"] = workersMax
			}
			if cmd.Flags().Changed("idle-timeout") {
				input["idleTimeout"] = idleTimeout
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "endpoint_update",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: EndpointInput!) {
				saveEndpoint(input: $input) { %s }
			}`, endpointFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var ep api.Endpoint
			for _, v := range result {
				if err := json.Unmarshal(v, &ep); err == nil && ep.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), ep)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Endpoint name")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Template ID")
	cmd.Flags().StringVar(&gpuIDs, "gpu-ids", "", "GPU type IDs")
	cmd.Flags().IntVar(&workersMin, "workers-min", 0, "Minimum workers")
	cmd.Flags().IntVar(&workersMax, "workers-max", 0, "Maximum workers")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")

	return cmd
}

func newEndpointSwapTemplateCmd() *cobra.Command {
	var templateID string

	cmd := &cobra.Command{
		Use:   "swap-template <id>",
		Short: "Swap the template assigned to an endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if templateID == "" {
				exitError("validation_error", "--template-id is required")
			}

			input := map[string]any{
				"endpointId": args[0],
				"templateId": templateID,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "endpoint_swap_template",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: UpdateEndpointTemplateInput!) {
				updateEndpointTemplate(input: $input) { %s }
			}`, endpointFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var ep api.Endpoint
			for _, v := range result {
				if err := json.Unmarshal(v, &ep); err == nil && ep.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), ep)
		},
	}

	cmd.Flags().StringVar(&templateID, "template-id", "", "New template ID (required)")

	return cmd
}

func newEndpointDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a serverless endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run":     true,
					"action":      "endpoint_delete",
					"endpoint_id": args[0],
				})
			}

			client := getClient()

			query := `mutation($id: String!) {
				deleteEndpoint(id: $id)
			}`

			vars := map[string]any{"id": args[0]}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Endpoint %s deleted", args[0]),
			})
		},
	}
}
