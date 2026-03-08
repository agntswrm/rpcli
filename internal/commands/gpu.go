package commands

import (
	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newGPUCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gpu",
		Short: "Manage GPU types and availability",
	}

	cmd.AddCommand(newGPUListCmd())
	cmd.AddCommand(newGPUAvailabilityCmd())

	return cmd
}

func newGPUListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available GPU types",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				gpuTypes {
					id
					displayName
					memoryInGb
					secureCloud
					communityCloud
				}
			}`

			var result struct {
				GPUTypes []api.GPUType `json:"gpuTypes"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.GPUTypes)
		},
	}
}

func newGPUAvailabilityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "availability",
		Short: "Show GPU availability and pricing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				gpuTypes {
					id
					displayName
					memoryInGb
					secureCloud
					communityCloud
					lowestPrice {
						minimumBidPrice
						uninterruptablePrice
					}
				}
			}`

			var result struct {
				GPUTypes []api.GPUType `json:"gpuTypes"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.GPUTypes)
		},
	}
}
