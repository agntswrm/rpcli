package commands

import (
	"encoding/json"
	"fmt"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const templateFields = `id name imageName dockerStartCmd containerDiskInGb volumeMountPath ports isPublic isServerless env { key value }`

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
		dockerCmd     string
		containerDisk float64
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

			input := map[string]any{
				"name":             name,
				"imageName":        image,
				"containerDiskInGb": containerDisk,
				"isServerless":     isServerless,
			}
			if dockerCmd != "" {
				input["dockerStartCmd"] = dockerCmd
			}
			if volumePath != "" {
				input["volumeMountPath"] = volumePath
			}
			if ports != "" {
				input["ports"] = ports
			}
			if len(envVars) > 0 {
				envList := parseEnvVars(envVars)
				input["env"] = envList
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "template_create",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: PodTemplateInput!) {
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
	cmd.Flags().StringVar(&dockerCmd, "docker-start-cmd", "", "Docker start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 20, "Container disk in GB")
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
		dockerCmd     string
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
			input := map[string]any{
				"id": args[0],
			}
			if cmd.Flags().Changed("name") {
				input["name"] = name
			}
			if cmd.Flags().Changed("image") {
				input["imageName"] = image
			}
			if cmd.Flags().Changed("docker-start-cmd") {
				input["dockerStartCmd"] = dockerCmd
			}
			if cmd.Flags().Changed("container-disk") {
				input["containerDiskInGb"] = containerDisk
			}
			if cmd.Flags().Changed("volume-path") {
				input["volumeMountPath"] = volumePath
			}
			if cmd.Flags().Changed("ports") {
				input["ports"] = ports
			}
			if cmd.Flags().Changed("env") {
				input["env"] = parseEnvVars(envVars)
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "template_update",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: PodTemplateInput!) {
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
	cmd.Flags().StringVar(&dockerCmd, "docker-start-cmd", "", "Docker start command")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 0, "Container disk in GB")
	cmd.Flags().StringVar(&volumePath, "volume-path", "", "Volume mount path")
	cmd.Flags().StringVar(&ports, "ports", "", "Ports to expose")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")

	return cmd
}

func newTemplateDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"templateId": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "template_delete",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: DeleteTemplateInput!) {
				deleteTemplate(input: $input)
			}`

			vars := map[string]any{"input": input}

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
