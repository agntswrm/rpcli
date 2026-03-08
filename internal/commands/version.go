package commands

import (
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of rpcli",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.Print(getFormat(), map[string]string{
				"version": Version,
			})
		},
	}
}
