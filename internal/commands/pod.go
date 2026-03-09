package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

const podFields = `
	id name imageName desiredStatus podType
	gpuCount volumeInGb containerDiskInGb memoryInGb vcpuCount
	costPerHr volumeMountPath ports dockerArgs templateId
	machineId uptimeSeconds locked createdAt lastStartedAt lastStatusChange
	env
`

func newPodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pod",
		Short: "Manage Runpod pods",
	}

	cmd.AddCommand(newPodListCmd())
	cmd.AddCommand(newPodGetCmd())
	cmd.AddCommand(newPodCreateCmd())
	cmd.AddCommand(newPodUpdateCmd())
	cmd.AddCommand(newPodStartCmd())
	cmd.AddCommand(newPodBidResumeCmd())
	cmd.AddCommand(newPodStopCmd())
	cmd.AddCommand(newPodRestartCmd())
	cmd.AddCommand(newPodResetCmd())
	cmd.AddCommand(newPodDeleteCmd())

	return cmd
}

func newPodListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all pods",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query { myself { pods { %s } } }`, podFields)

			var result struct {
				Myself struct {
					Pods []api.Pod `json:"pods"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.Pods)
		},
	}
}

func newPodGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get details of a specific pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := fmt.Sprintf(`query($input: PodFilter!) { pod(input: $input) { %s } }`, podFields)
			vars := map[string]any{
				"input": map[string]any{"podId": args[0]},
			}

			var result struct {
				Pod api.Pod `json:"pod"`
			}
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Pod)
		},
	}
}

func newPodCreateCmd() *cobra.Command {
	var (
		name            string
		gpuCount        int
		image           string
		volumeSize      float64
		containerDisk   float64
		templateID      string
		envVars         []string
		ports           string
		volumePath      string
		cloudType       string
		spot            bool
		bidPrice        float64
		networkVolumeID string
	)

	cmd := &cobra.Command{
		Use:   "create <hardware>",
		Short: "Create a new pod (auto-detects GPU vs CPU)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hardware := args[0]
			if image == "" && templateID == "" {
				exitError("validation_error", "--image or --template-id is required")
			}
			if spot && bidPrice <= 0 {
				exitError("validation_error", "--bid-price is required when using --spot (set your max $/hr per GPU)")
			}

			client := getClient()

			// Detect whether hardware is a CPU or GPU type
			isCPU := isHardwareCPU(client, hardware)

			// Build shared input fields
			input := map[string]any{}
			if name != "" {
				input["name"] = name
			}
			if image != "" {
				input["imageName"] = image
			}
			if volumeSize > 0 {
				input["volumeInGb"] = volumeSize
			}
			if containerDisk > 0 {
				input["containerDiskInGb"] = containerDisk
			}
			if templateID != "" {
				input["templateId"] = templateID
			}
			if ports != "" {
				input["ports"] = ports
			}
			if volumePath != "" {
				input["volumeMountPath"] = volumePath
			}
			if len(envVars) > 0 {
				input["env"] = envVars
			}
			if networkVolumeID != "" {
				input["networkVolumeId"] = networkVolumeID
			}

			var query string
			if isCPU {
				input["instanceId"] = hardware

				if dryRunFlag {
					return output.Print(getFormat(), map[string]any{
						"dry_run":  true,
						"action":   "pod_create",
						"mutation": "deployCpuPod",
						"input":    input,
					})
				}

				query = fmt.Sprintf(`mutation($input: deployCpuPodInput!) {
					deployCpuPod(input: $input) { %s }
				}`, podFields)
			} else {
				input["gpuTypeId"] = hardware
				input["gpuCount"] = gpuCount
				input["cloudType"] = cloudType

				mutationName := "podFindAndDeployOnDemand"
				inputType := "PodFindAndDeployOnDemandInput"
				if spot {
					mutationName = "podRentInterruptable"
					inputType = "PodRentInterruptableInput"
					input["bidPerGpu"] = bidPrice
				}

				if dryRunFlag {
					return output.Print(getFormat(), map[string]any{
						"dry_run":  true,
						"action":   "pod_create",
						"mutation": mutationName,
						"input":    input,
					})
				}

				query = fmt.Sprintf(`mutation($input: %s!) {
					%s(input: $input) { %s }
				}`, inputType, mutationName, podFields)
			}

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Pod name")
	cmd.Flags().IntVar(&gpuCount, "gpu-count", 1, "Number of GPUs (GPU pods only)")
	cmd.Flags().StringVar(&image, "image", "", "Docker image name")
	cmd.Flags().Float64Var(&volumeSize, "volume-size", 0, "Volume size in GB")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 20, "Container disk size in GB")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Template ID to use")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVar(&ports, "ports", "", "Ports to expose (e.g. '8888/http,22/tcp')")
	cmd.Flags().StringVar(&volumePath, "volume-path", "/workspace", "Volume mount path")
	cmd.Flags().StringVar(&cloudType, "cloud-type", "ALL", "Cloud type: ALL, SECURE, COMMUNITY (GPU pods only)")
	cmd.Flags().BoolVar(&spot, "spot", false, "Create a spot/interruptable pod (GPU pods only, requires --bid-price)")
	cmd.Flags().Float64Var(&bidPrice, "bid-price", 0, "Max bid price per GPU in $/hr (required with --spot)")
	cmd.Flags().StringVar(&networkVolumeID, "network-volume", "", "Network volume ID to attach")

	return cmd
}

// isHardwareCPU queries the API to determine if a hardware ID is a CPU flavor or GPU type.
// Returns true for CPU (cpuFlavor), false for GPU. Exits on error if the ID is not found in either list.
func isHardwareCPU(client *api.Client, hardwareID string) bool {
	// Check GPU types first (more common)
	var gpuResult struct {
		GPUTypes []struct {
			ID string `json:"id"`
		} `json:"gpuTypes"`
	}
	gpuQuery := `query { gpuTypes { id } }`
	if err := client.Execute(gpuQuery, nil, &gpuResult); err != nil {
		exitError("api_error", fmt.Sprintf("failed to query hardware types: %s", err.Error()))
	}
	for _, g := range gpuResult.GPUTypes {
		if g.ID == hardwareID {
			return false
		}
	}

	// Check CPU flavors (used as instanceId for deployCpuPod)
	// instanceId format: <flavor>-<vcpus>-<memoryGB> (e.g., "cpu3c-2-4")
	// Accept exact match on flavor ID or prefix match with dash separator
	var cpuResult struct {
		CPUFlavors []struct {
			ID string `json:"id"`
		} `json:"cpuFlavors"`
	}
	cpuQuery := `query { cpuFlavors { id } }`
	if err := client.Execute(cpuQuery, nil, &cpuResult); err != nil {
		exitError("api_error", fmt.Sprintf("failed to query hardware types: %s", err.Error()))
	}
	for _, c := range cpuResult.CPUFlavors {
		if c.ID == hardwareID || strings.HasPrefix(hardwareID, c.ID+"-") {
			return true
		}
	}

	exitError("validation_error", fmt.Sprintf("hardware ID %q not found in GPU types or CPU flavors (use 'resource gpu' or 'resource cpu' to list valid IDs)", hardwareID))
	return false
}

func newPodUpdateCmd() *cobra.Command {
	var (
		gpuCount      int
		volumeSize    float64
		containerDisk float64
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := map[string]any{
				"podId": args[0],
			}
			if cmd.Flags().Changed("gpu-count") {
				input["gpuCount"] = gpuCount
			}
			if cmd.Flags().Changed("volume-size") {
				input["volumeInGb"] = volumeSize
			}
			if cmd.Flags().Changed("container-disk") {
				input["containerDiskInGb"] = containerDisk
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_update",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: PodEditJobInput!) {
				podEditJob(input: $input) { %s }
			}`, podFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}

	cmd.Flags().IntVar(&gpuCount, "gpu-count", 0, "Number of GPUs")
	cmd.Flags().Float64Var(&volumeSize, "volume-size", 0, "Volume size in GB")
	cmd.Flags().Float64Var(&containerDisk, "container-disk", 0, "Container disk size in GB")

	return cmd
}

func newPodStartCmd() *cobra.Command {
	var gpuCount int

	cmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Start a stopped pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := map[string]any{
				"podId":    args[0],
				"gpuCount": gpuCount,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_start",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: PodResumeInput!) {
				podResume(input: $input) { %s }
			}`, podFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}

	cmd.Flags().IntVar(&gpuCount, "gpu-count", 1, "Number of GPUs")

	return cmd
}

func newPodBidResumeCmd() *cobra.Command {
	var (
		gpuCount int
		bidPrice float64
	)

	cmd := &cobra.Command{
		Use:   "bid-resume <id>",
		Short: "Resume a spot/interruptible pod with a bid price",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if bidPrice <= 0 {
				exitError("validation_error", "--bid-price is required and must be positive")
			}

			input := map[string]any{
				"podId":     args[0],
				"gpuCount":  gpuCount,
				"bidPerGpu": bidPrice,
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_bid_resume",
					"input":   input,
				})
			}

			client := getClient()

			query := fmt.Sprintf(`mutation($input: PodBidResumeInput!) {
				podBidResume(input: $input) { %s }
			}`, podFields)

			vars := map[string]any{"input": input}

			var result map[string]json.RawMessage
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}

	cmd.Flags().IntVar(&gpuCount, "gpu-count", 1, "Number of GPUs")
	cmd.Flags().Float64Var(&bidPrice, "bid-price", 0, "Bid price per GPU (required)")

	return cmd
}

func newPodStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a running pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"podId": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_stop",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: PodStopInput!) {
				podStop(input: $input) { id desiredStatus }
			}`

			vars := map[string]any{"input": input}

			var result struct {
				PodStop api.Pod `json:"podStop"`
			}
			if err := client.Execute(query, vars, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"id":      result.PodStop.ID,
				"message": fmt.Sprintf("Pod %s stopped", args[0]),
			})
		},
	}
}

func newPodRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"podId": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_restart",
					"input":   input,
				})
			}

			client := getClient()

			// Stop
			stopQuery := `mutation($input: PodStopInput!) {
				podStop(input: $input) { id desiredStatus }
			}`
			if err := client.Execute(stopQuery, map[string]any{"input": input}, nil); err != nil {
				exitError("api_error", err.Error())
			}

			// Start
			startQuery := fmt.Sprintf(`mutation($input: PodResumeInput!) {
				podResume(input: $input) { %s }
			}`, podFields)
			startInput := map[string]any{"podId": args[0], "gpuCount": 1}

			var result map[string]json.RawMessage
			if err := client.Execute(startQuery, map[string]any{"input": startInput}, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}
}

func newPodResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <id>",
		Short: "Reset a pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"podId": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_reset",
					"input":   input,
				})
			}

			client := getClient()

			// Stop then start to reset
			stopQuery := `mutation($input: PodStopInput!) {
				podStop(input: $input) { id desiredStatus }
			}`
			if err := client.Execute(stopQuery, map[string]any{"input": input}, nil); err != nil {
				exitError("api_error", err.Error())
			}

			startQuery := fmt.Sprintf(`mutation($input: PodResumeInput!) {
				podResume(input: $input) { %s }
			}`, podFields)
			startInput := map[string]any{"podId": args[0], "gpuCount": 1}

			var result map[string]json.RawMessage
			if err := client.Execute(startQuery, map[string]any{"input": startInput}, &result); err != nil {
				exitError("api_error", err.Error())
			}

			var pod api.Pod
			for _, v := range result {
				if err := json.Unmarshal(v, &pod); err == nil && pod.ID != "" {
					break
				}
			}

			return output.Print(getFormat(), pod)
		},
	}
}

func newPodDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a pod permanently",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yesFlag {
				exitError("confirmation_required", "This is a destructive operation. Use --yes to confirm.")
			}

			input := map[string]any{"podId": args[0]}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run": true,
					"action":  "pod_delete",
					"input":   input,
				})
			}

			client := getClient()

			query := `mutation($input: PodTerminateInput!) {
				podTerminate(input: $input)
			}`

			vars := map[string]any{"input": input}

			if err := client.Execute(query, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": fmt.Sprintf("Pod %s deleted", args[0]),
			})
		},
	}
}
