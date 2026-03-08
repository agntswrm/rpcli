package commands

import (
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newBillingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "View billing and spend information",
	}

	cmd.AddCommand(newBillingPodsCmd())
	cmd.AddCommand(newBillingEndpointsCmd())
	cmd.AddCommand(newBillingVolumesCmd())

	return cmd
}

func newBillingPodsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pods",
		Short: "View pod billing information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				myself {
					pods {
						id name costPerHr uptimeSeconds
					}
				}
			}`

			var result struct {
				Myself struct {
					Pods []struct {
						ID            string  `json:"id"`
						Name          string  `json:"name"`
						CostPerHr     float64 `json:"costPerHr"`
						UptimeSeconds int     `json:"uptimeSeconds"`
					} `json:"pods"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			// Calculate estimated costs
			type podBilling struct {
				ID            string  `json:"id"`
				Name          string  `json:"name"`
				CostPerHr     float64 `json:"cost_per_hr"`
				UptimeSeconds int     `json:"uptime_seconds"`
				EstimatedCost float64 `json:"estimated_cost"`
			}

			items := make([]podBilling, len(result.Myself.Pods))
			for i, p := range result.Myself.Pods {
				items[i] = podBilling{
					ID:            p.ID,
					Name:          p.Name,
					CostPerHr:     p.CostPerHr,
					UptimeSeconds: p.UptimeSeconds,
					EstimatedCost: p.CostPerHr * float64(p.UptimeSeconds) / 3600.0,
				}
			}

			return output.Print(getFormat(), items)
		},
	}
}

func newBillingEndpointsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "endpoints",
		Short: "View endpoint billing information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				myself {
					endpoints {
						id name workersMin workersMax
					}
				}
			}`

			var result struct {
				Myself struct {
					Endpoints []struct {
						ID         string `json:"id"`
						Name       string `json:"name"`
						WorkersMin int    `json:"workersMin"`
						WorkersMax int    `json:"workersMax"`
					} `json:"endpoints"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.Endpoints)
		},
	}
}

func newBillingVolumesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "volumes",
		Short: "View volume billing information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				myself {
					networkVolumes {
						id name size dataCenterId
					}
				}
			}`

			var result struct {
				Myself struct {
					NetworkVolumes []struct {
						ID           string  `json:"id"`
						Name         string  `json:"name"`
						Size         float64 `json:"size"`
						DataCenterID string  `json:"dataCenterId"`
					} `json:"networkVolumes"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.Myself.NetworkVolumes)
		},
	}
}
