package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const endpointFields = `id name templateId gpuIds workersMin workersMax idleTimeout networkVolumeId`

// autoTemplatePrefix is used to identify templates auto-created by endpoint commands.
const autoTemplatePrefix = "rpcli-ep-"

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

// createServerlessTemplate creates a serverless template and returns its ID.
func createServerlessTemplate(client *api.Client, name, image, dockerArgs string, containerDisk float64, volumeSize int, volumePath string, envVars []string) (string, error) {
	envList := parseEnvList(envVars)

	suffix := fmt.Sprintf("%d", time.Now().Unix())
	input := map[string]any{
		"name":              autoTemplatePrefix + name + "-" + suffix,
		"imageName":         image,
		"containerDiskInGb": containerDisk,
		"volumeInGb":        volumeSize,
		"isServerless":      true,
		"dockerArgs":        dockerArgs,
		"env":               envList,
	}
	if volumePath != "" {
		input["volumeMountPath"] = volumePath
	}

	query := `mutation($input: SaveTemplateInput!) {
		saveTemplate(input: $input) { id name }
	}`

	var result map[string]json.RawMessage
	if err := client.Execute(query, map[string]any{"input": input}, &result); err != nil {
		return "", fmt.Errorf("failed to create template: %w", err)
	}

	var tmpl api.Template
	for _, v := range result {
		if err := json.Unmarshal(v, &tmpl); err == nil && tmpl.ID != "" {
			break
		}
	}
	if tmpl.ID == "" {
		return "", fmt.Errorf("failed to create template: empty response")
	}

	return tmpl.ID, nil
}

// parseEnvList converts KEY=VALUE strings to EnvironmentVariableInput format.
func parseEnvList(envVars []string) []map[string]string {
	envList := make([]map[string]string, 0, len(envVars))
	for _, e := range envVars {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envList = append(envList, map[string]string{"key": parts[0], "value": parts[1]})
		}
	}
	return envList
}

func newEndpointCreateCmd() *cobra.Command {
	var (
		name            string
		image           string
		gpuIDs          string
		workersMin      int
		workersMax      int
		idleTimeout     int
		templateID      string
		dockerArgs      string
		containerDisk   float64
		volumeSize      int
		volumePath      string
		envVars         []string
		networkVolumeID string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a serverless endpoint (auto-creates template from --image)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || gpuIDs == "" {
				exitError("validation_error", "--name and --gpus are required")
			}
			if image == "" && templateID == "" {
				exitError("validation_error", "--image or --template-id is required")
			}

			if dryRunFlag {
				dryOut := map[string]any{
					"dry_run": true,
					"action":  "endpoint_create",
					"endpoint": map[string]any{
						"name":       name,
						"gpuIds":     gpuIDs,
						"workersMin": workersMin,
						"workersMax": workersMax,
					},
				}
				if templateID != "" {
					dryOut["template_id"] = templateID
				} else {
					dryOut["auto_template"] = map[string]any{
						"name":  autoTemplatePrefix + name + "-<timestamp>",
						"image": image,
						"env":   envVars,
					}
				}
				return output.Print(getFormat(), dryOut)
			}

			client := getClient()

			// Auto-create serverless template if no template ID given
			if templateID == "" {
				id, err := createServerlessTemplate(client, name, image, dockerArgs, containerDisk, volumeSize, volumePath, envVars)
				if err != nil {
					exitError("api_error", err.Error())
				}
				templateID = id
			}

			// Create the endpoint
			epInput := map[string]any{
				"name":       name,
				"templateId": templateID,
				"gpuIds":     gpuIDs,
				"workersMin": workersMin,
				"workersMax": workersMax,
			}
			if idleTimeout > 0 {
				epInput["idleTimeout"] = idleTimeout
			}
			if networkVolumeID != "" {
				epInput["networkVolumeId"] = networkVolumeID
			}

			query := fmt.Sprintf(`mutation($input: EndpointInput!) {
				saveEndpoint(input: $input) { %s }
			}`, endpointFields)

			var result map[string]json.RawMessage
			if err := client.Execute(query, map[string]any{"input": epInput}, &result); err != nil {
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
	cmd.Flags().StringVar(&image, "image", "", "Docker image (creates serverless template automatically)")
	cmd.Flags().StringVar(&gpuIDs, "gpus", "", "GPU type IDs (required)")
	cmd.Flags().IntVar(&workersMin, "workers-min", 0, "Minimum number of workers")
	cmd.Flags().IntVar(&workersMax, "workers-max", 1, "Maximum number of workers")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Use an existing template instead of --image")
	cmd.Flags().StringVar(&dockerArgs, "docker-args", "", "Docker arguments/start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 20, "Container disk in GB")
	cmd.Flags().IntVar(&volumeSize, "volume-size", 0, "Persistent volume size in GB")
	cmd.Flags().StringVar(&volumePath, "volume-path", "", "Volume mount path")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVar(&networkVolumeID, "network-volume", "", "Network volume ID to attach")

	return cmd
}

func newEndpointUpdateCmd() *cobra.Command {
	var (
		name            string
		image           string
		gpuIDs          string
		workersMin      int
		workersMax      int
		idleTimeout     int
		dockerArgs      string
		containerDisk   float64
		volumeSize      int
		volumePath      string
		envVars         []string
		networkVolumeID string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a serverless endpoint (and its template if --image/--env/--docker-args are changed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateChanged := cmd.Flags().Changed("image") || cmd.Flags().Changed("env") ||
				cmd.Flags().Changed("docker-args") || cmd.Flags().Changed("container-disk") ||
				cmd.Flags().Changed("volume-size") || cmd.Flags().Changed("volume-path")
			endpointChanged := cmd.Flags().Changed("name") || cmd.Flags().Changed("gpus") ||
				cmd.Flags().Changed("workers-min") || cmd.Flags().Changed("workers-max") ||
				cmd.Flags().Changed("idle-timeout") || cmd.Flags().Changed("network-volume")

			if !endpointChanged && !templateChanged {
				exitError("validation_error", "no flags specified to update")
			}

			client := getClient()

			// Always fetch current endpoint (needed for both paths)
			listQuery := fmt.Sprintf(`query { myself { endpoints { %s } } }`, endpointFields)
			var listResult struct {
				Myself struct {
					Endpoints []api.Endpoint `json:"endpoints"`
				} `json:"myself"`
			}
			if err := client.Execute(listQuery, nil, &listResult); err != nil {
				exitError("api_error", err.Error())
			}

			var current *api.Endpoint
			for i, ep := range listResult.Myself.Endpoints {
				if ep.ID == args[0] {
					current = &listResult.Myself.Endpoints[i]
					break
				}
			}
			if current == nil {
				exitError("not_found", fmt.Sprintf("Endpoint %s not found", args[0]))
			}

			// Update template if needed
			if templateChanged {
				tmplQuery := `query { myself { podTemplates { id name imageName containerDiskInGb isServerless } } }`
				var tmplResult struct {
					Myself struct {
						PodTemplates []api.Template `json:"podTemplates"`
					} `json:"myself"`
				}
				if err := client.Execute(tmplQuery, nil, &tmplResult); err != nil {
					exitError("api_error", err.Error())
				}

				var tmpl *api.Template
				for i, t := range tmplResult.Myself.PodTemplates {
					if t.ID == current.TemplateID {
						tmpl = &tmplResult.Myself.PodTemplates[i]
						break
					}
				}
				if tmpl == nil {
					exitError("not_found", fmt.Sprintf("Template %s for endpoint not found", current.TemplateID))
				}

				tmplInput := map[string]any{
					"id":                tmpl.ID,
					"name":              tmpl.Name,
					"imageName":         tmpl.ImageName,
					"containerDiskInGb": int(tmpl.ContainerDisk),
					"isServerless":      true,
					"dockerArgs":        "",
					"env":               []map[string]string{},
					"volumeInGb":        0,
				}
				if cmd.Flags().Changed("image") {
					tmplInput["imageName"] = image
				}
				if cmd.Flags().Changed("docker-args") {
					tmplInput["dockerArgs"] = dockerArgs
				}
				if cmd.Flags().Changed("container-disk") {
					tmplInput["containerDiskInGb"] = int(containerDisk)
				}
				if cmd.Flags().Changed("volume-size") {
					tmplInput["volumeInGb"] = volumeSize
				}
				if cmd.Flags().Changed("volume-path") {
					tmplInput["volumeMountPath"] = volumePath
				}
				if cmd.Flags().Changed("env") {
					tmplInput["env"] = parseEnvList(envVars)
				}

				if dryRunFlag && !endpointChanged {
					return output.Print(getFormat(), map[string]any{
						"dry_run":         true,
						"action":          "endpoint_update",
						"endpoint_id":     args[0],
						"template_update": tmplInput,
					})
				}

				tmplMutation := `mutation($input: SaveTemplateInput!) {
					saveTemplate(input: $input) { id name }
				}`
				if err := client.Execute(tmplMutation, map[string]any{"input": tmplInput}, nil); err != nil {
					exitError("api_error", fmt.Sprintf("failed to update template: %s", err.Error()))
				}
			}

			// Update endpoint if needed (saveEndpoint is upsert, needs all fields)
			if endpointChanged {
				epInput := map[string]any{
					"id":         current.ID,
					"name":       current.Name,
					"templateId": current.TemplateID,
					"gpuIds":     current.GPUIDs,
					"workersMin": current.WorkersMin,
					"workersMax": current.WorkersMax,
					"idleTimeout": current.IdleTimeout,
				}
				if current.NetworkVolume != "" {
					epInput["networkVolumeId"] = current.NetworkVolume
				}
				if cmd.Flags().Changed("name") {
					epInput["name"] = name
				}
				if cmd.Flags().Changed("gpus") {
					epInput["gpuIds"] = gpuIDs
				}
				if cmd.Flags().Changed("workers-min") {
					epInput["workersMin"] = workersMin
				}
				if cmd.Flags().Changed("workers-max") {
					epInput["workersMax"] = workersMax
				}
				if cmd.Flags().Changed("idle-timeout") {
					epInput["idleTimeout"] = idleTimeout
				}
				if cmd.Flags().Changed("network-volume") {
					epInput["networkVolumeId"] = networkVolumeID
				}

				if dryRunFlag {
					return output.Print(getFormat(), map[string]any{
						"dry_run":     true,
						"action":      "endpoint_update",
						"endpoint_id": args[0],
						"input":       epInput,
					})
				}

				query := fmt.Sprintf(`mutation($input: EndpointInput!) {
					saveEndpoint(input: $input) { %s }
				}`, endpointFields)

				var result map[string]json.RawMessage
				if err := client.Execute(query, map[string]any{"input": epInput}, &result); err != nil {
					exitError("api_error", err.Error())
				}

				var ep api.Endpoint
				for _, v := range result {
					if err := json.Unmarshal(v, &ep); err == nil && ep.ID != "" {
						break
					}
				}
				return output.Print(getFormat(), ep)
			}

			// Template-only update: return the current endpoint
			return output.Print(getFormat(), *current)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Endpoint name")
	cmd.Flags().StringVar(&image, "image", "", "Docker image (updates the endpoint's template)")
	cmd.Flags().StringVar(&gpuIDs, "gpus", "", "GPU type IDs")
	cmd.Flags().IntVar(&workersMin, "workers-min", 0, "Minimum workers")
	cmd.Flags().IntVar(&workersMax, "workers-max", 0, "Maximum workers")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout in seconds")
	cmd.Flags().StringVar(&dockerArgs, "docker-args", "", "Docker arguments/start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 0, "Container disk in GB")
	cmd.Flags().IntVar(&volumeSize, "volume-size", 0, "Persistent volume size in GB")
	cmd.Flags().StringVar(&volumePath, "volume-path", "", "Volume mount path")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVar(&networkVolumeID, "network-volume", "", "Network volume ID to attach")

	return cmd
}

func newEndpointDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a serverless endpoint (auto-cleans up auto-created templates)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			client := getClient()

			// Fetch endpoint to find its template
			listQuery := fmt.Sprintf(`query { myself { endpoints { %s } } }`, endpointFields)
			var listResult struct {
				Myself struct {
					Endpoints []api.Endpoint `json:"endpoints"`
				} `json:"myself"`
			}
			if err := client.Execute(listQuery, nil, &listResult); err != nil {
				exitError("api_error", err.Error())
			}

			var templateID string
			for _, ep := range listResult.Myself.Endpoints {
				if ep.ID == args[0] {
					templateID = ep.TemplateID
					break
				}
			}

			if dryRunFlag {
				dryOut := map[string]any{
					"dry_run":     true,
					"action":      "endpoint_delete",
					"endpoint_id": args[0],
				}
				if templateID != "" {
					dryOut["template_cleanup"] = templateID
				}
				return output.Print(getFormat(), dryOut)
			}

			// Delete the endpoint
			delQuery := `mutation($id: String!) {
				deleteEndpoint(id: $id)
			}`
			if err := client.Execute(delQuery, map[string]any{"id": args[0]}, nil); err != nil {
				exitError("api_error", err.Error())
			}

			// Clean up auto-created template if applicable
			templateCleaned := false
			if templateID != "" {
				// Check if the template was auto-created by looking for our prefix
				tmplQuery := `query { myself { podTemplates { id name } } }`
				var tmplResult struct {
					Myself struct {
						PodTemplates []api.Template `json:"podTemplates"`
					} `json:"myself"`
				}
				if err := client.Execute(tmplQuery, nil, &tmplResult); err == nil {
					for _, t := range tmplResult.Myself.PodTemplates {
						if t.ID == templateID && strings.HasPrefix(t.Name, autoTemplatePrefix) {
							delTmpl := `mutation($name: String!) { deleteTemplate(templateName: $name) }`
							if err := client.Execute(delTmpl, map[string]any{"name": t.Name}, nil); err == nil {
								templateCleaned = true
							}
							break
						}
					}
				}
			}

			result := map[string]any{
				"status":           "ok",
				"message":          fmt.Sprintf("Endpoint %s deleted", args[0]),
				"template_cleaned": templateCleaned,
			}

			return output.Print(getFormat(), result)
		},
	}
}
