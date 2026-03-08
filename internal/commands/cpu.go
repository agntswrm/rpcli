package commands

import (
	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newCPUCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cpu",
		Short: "Manage CPU types",
	}

	cmd.AddCommand(newCPUListCmd())

	return cmd
}

func newCPUListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available CPU types",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				cpuTypes {
					id
					displayName
					manufacturer
					cores
					threadsPerCore
					groupId
				}
			}`

			var result struct {
				CPUTypes []api.CPUType `json:"cpuTypes"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), result.CPUTypes)
		},
	}
}
