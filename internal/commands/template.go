package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const templateFields = `id name imageName containerDiskInGb isServerless`

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage pod templates",
	}

	cmd.AddCommand(newTemplateListCmd())
	cmd.AddCommand(newTemplateCreateCmd())
	cmd.AddCommand(newTemplateUpdateCmd())
	cmd.AddCommand(newTemplateDeleteCmd())

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { podTemplates { %s } } }`, templateFields)

			var result struct {
				Myself struct {
					PodTemplates []api.Template `json:"podTemplates"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.PodTemplates)
		},
	}
}

func newTemplateCreateCmd() *cobra.Command {
	var (
		name          string
		image         string
		dockerArgs    string
		containerDisk float64
		volumeSize    int
		volumePath    string
		ports         string
		isServerless  bool
		envVars       []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pod template",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || image == "" {
				exitError("validation_error", "--name and --image are required")
			}

			// Build env variable list in EnvironmentVariableInput format
			envList := make([]map[string]string, 0, len(envVars))
			for _, e := range envVars {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envList = append(envList, map[string]string{"key": parts[0], "value": parts[1]})
				}
			}

			input := map[string]any{
				"name":             name,
				"imageName":        image,
				"containerDiskInGb": containerDisk,
				"volumeInGb":       volumeSize,
				"isServerless":     isServerless,
				"dockerArgs":       dockerArgs,
				"env":              envList,
			}
			if volumePath != "" {
				input["volumeMountPath"] = volumePath
			}
			if ports != "" {
				input["ports"] = ports
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "template_create",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: SaveTemplateInput!) {
				saveTemplate(input: $input) { %s }
			}`, templateFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var tmpl api.Template
			for _, v := range result {
				if err := json.Unmarshal(v, &tmpl); err == nil && tmpl.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), tmpl)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Template name (required)")
	cmd.Flags().StringVar(&image, "image", "", "Docker image (required)")
	cmd.Flags().StringVar(&dockerArgs, "docker-args", "", "Docker arguments/start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 20, "Container disk in GB")
	cmd.Flags().IntVar(&volumeSize, "volume-size", 0, "Persistent volume size in GB")
	cmd.Flags().StringVar(&volumePath, "volume-path", "", "Volume mount path")
	cmd.Flags().StringVar(&ports, "ports", "", "Ports to expose")
	cmd.Flags().BoolVar(&isServerless, "serverless", false, "Create as serverless template")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")

	return cmd
}

func newTemplateUpdateCmd() *cobra.Command {
	var (
		name          string
		image         string
		dockerArgs    string
		containerDisk float64
		volumePath    string
		ports         string
		envVars       []string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a pod template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			// Fetch current template to merge changes (saveTemplate is an upsert requiring all fields)
			listQuery := `query { myself { podTemplates { id name imageName containerDiskInGb isServerless } } }`
			var listResult struct {
				Myself struct {
					PodTemplates []api.Template `json:"podTemplates"`
				} `json:"myself"`
			}
			if err := client.Execute(listQuery, nil, &listResult); err != nil {
				exitError("api_error", err.Error())
			}

			var current *api.Template
			for i, t := range listResult.Myself.PodTemplates {
				if t.ID == args[0] {
					current = &listResult.Myself.PodTemplates[i]
					break
				}
			}
			if current == nil {
				exitError("not_found", fmt.Sprintf("Template %s not found", args[0]))
			}

			// Start with current values
			input := map[string]any{
				"id":               current.ID,
				"name":             current.Name,
				"imageName":        current.ImageName,
				"containerDiskInGb": int(current.ContainerDisk),
				"isServerless":     current.IsServerless,
				"dockerArgs":       "",
				"env":              []map[string]string{},
				"volumeInGb":       0,
			}

			// Apply overrides
			if cmd.Flags().Changed("name") {
				input["name"] = name
			}
			if cmd.Flags().Changed("image") {
				input["imageName"] = image
			}
			if cmd.Flags().Changed("docker-args") {
				input["dockerArgs"] = dockerArgs
			}
			if cmd.Flags().Changed("container-disk") {
				input["containerDiskInGb"] = int(containerDisk)
			}
			if cmd.Flags().Changed("volume-path") {
				input["volumeMountPath"] = volumePath
			}
			if cmd.Flags().Changed("ports") {
				input["ports"] = ports
			}
			if cmd.Flags().Changed("env") {
				envList := make([]map[string]string, 0, len(envVars))
				for _, e := range envVars {
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						envList = append(envList, map[string]string{"key": parts[0], "value": parts[1]})
					}
				}
				input["env"] = envList
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "template_update",
					"input":   input,
				})
			}

			query := fmt.Sprintf(`mutation($input: SaveTemplateInput!) {
				saveTemplate(input: $input) { %s }
			}`, templateFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var tmpl api.Template
			for _, v := range result {
				if err := json.Unmarshal(v, &tmpl); err == nil && tmpl.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), tmpl)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Template name")
	cmd.Flags().StringVar(&image, "image", "", "Docker image")
	cmd.Flags().StringVar(&dockerArgs, "docker-args", "", "Docker arguments/start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 0, "Container disk in GB")
	cmd.Flags().StringVar(&volumePath, "volume-path", "", "Volume mount path")
	cmd.Flags().StringVar(&ports, "ports", "", "Ports to expose")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")

	return cmd
}

func newTemplateDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a template by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run":       true,
					"action":        "template_delete",
					"template_name": args[0],
				})
			}

			client := getClient()

			query := `mutation($name: String!) {
				deleteTemplate(templateName: $name)
			}`

			vars := map[string]any{"name": args[0]}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Template %s deleted", args[0]),
			})
		},
	}
}
