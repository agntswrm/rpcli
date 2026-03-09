package commands

import (
	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "Browse available GPUs and CPUs",
	}

	cmd.AddCommand(newResourceGPUCmd())
	cmd.AddCommand(newResourceCPUCmd())
	cmd.AddCommand(newResourceAvailabilityCmd())

	return cmd
}

// gpuListEntry is a slim output type for the gpu list (no pricing columns).
type gpuListEntry struct {
	ID             string `json:"id" yaml:"id"`
	DisplayName    string `json:"displayName" yaml:"displayName"`
	MemoryInGB     int    `json:"memoryInGb" yaml:"memoryInGb"`
	StockStatus    string `json:"stockStatus" yaml:"stockStatus"`
}

func newResourceGPUCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gpu",
		Short: "List available GPU types (secure cloud, in stock)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				gpuTypes {
					id
					displayName
					memoryInGb
					secureCloud
				}
			}`

			var result struct {
				GPUTypes []api.GPUType `json:"gpuTypes"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			stock := queryStockStatus(client)

			var filtered []gpuListEntry
			for _, g := range result.GPUTypes {
				if !g.SecureCloud {
					continue
				}
				s, inStock := stock[g.ID]
				if !inStock || s == "" {
					continue
				}
				filtered = append(filtered, gpuListEntry{
					ID:          g.ID,
					DisplayName: g.DisplayName,
					MemoryInGB:  g.MemoryInGB,
					StockStatus: s,
				})
			}

			return output.Print(getFormat(), filtered)
		},
	}
}

func newResourceCPUCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cpu",
		Short: "List available CPU flavors for pod creation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				cpuFlavors {
					id
					groupId
				}
			}`

			var result struct {
				CPUFlavors []api.CPUFlavor `json:"cpuFlavors"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.CPUFlavors)
		},
	}
}

func newResourceAvailabilityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "availability [name]",
		Short: "Show GPU availability, pricing, and stock (optionally filter by name)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			gpuQuery := `query {
				gpuTypes {
					id
					displayName
					memoryInGb
					secureCloud
					communityCloud
					securePrice
					communityPrice
					secureSpotPrice
					communitySpotPrice
					maxGpuCount
				}
			}`

			var gpuResult struct {
				GPUTypes []api.GPUType `json:"gpuTypes"`
			}
			if err := client.Execute(gpuQuery, nil, &gpuResult); err != nil {
				exitError("api_error", err.Error())
			}

			stock := queryStockStatus(client)

			// Filter to secure cloud + in stock
			var filtered []api.GPUType
			for i := range gpuResult.GPUTypes {
				g := &gpuResult.GPUTypes[i]
				if !g.SecureCloud {
					continue
				}
				s, inStock := stock[g.ID]
				if !inStock || s == "" {
					continue
				}
				g.StockStatus = s
				filtered = append(filtered, *g)
			}

			if len(args) == 1 {
				for _, g := range filtered {
					if g.ID == args[0] {
						return output.Print(getFormat(), g)
					}
				}
				exitError("not_found", "Resource "+args[0]+" not found")
			}

			return output.Print(getFormat(), filtered)
		},
	}
}

// queryStockStatus returns best stock status per GPU across all datacenters.
func queryStockStatus(client *api.Client) map[string]string {
	stockQuery := `query {
		dataCenters {
			gpuAvailability {
				gpuTypeId
				stockStatus
			}
		}
	}`

	var stockResult struct {
		DataCenters []struct {
			GPUAvailability []struct {
				GPUTypeID   string `json:"gpuTypeId"`
				StockStatus string `json:"stockStatus"`
			} `json:"gpuAvailability"`
		} `json:"dataCenters"`
	}
	// Best-effort — don't fail if this query errors
	_ = client.Execute(stockQuery, nil, &stockResult)

	rank := map[string]int{"High": 3, "Medium": 2, "Low": 1}
	best := make(map[string]string)
	for _, dc := range stockResult.DataCenters {
		for _, a := range dc.GPUAvailability {
			if rank[a.StockStatus] > rank[best[a.GPUTypeID]] {
				best[a.GPUTypeID] = a.StockStatus
			}
		}
	}
	return best
}
