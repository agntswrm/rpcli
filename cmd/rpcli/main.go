package main

import (
	"os"

	"github.com/agntswrm/rpcli/internal/commands"
	"github.com/agntswrm/rpcli/internal/output"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	commands.Version = version

	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		output.PrintError(output.FormatJSON, "command_error", err.Error())
		os.Exit(1)
	}
}
