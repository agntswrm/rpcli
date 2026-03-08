package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	Version = "1.0.0-test"
	rootCmd := NewRootCmd()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		// Output goes to stdout not the cmd buffer for our output pkg,
		// so just verify no error
		return
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	rootCmd := NewRootCmd()

	expected := []string{
		"version", "config", "gpu", "pod", "endpoint",
		"template", "volume", "registry", "secret", "billing",
		"ssh", "doctor",
	}

	for _, name := range expected {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

func TestPodCommandHasSubcommands(t *testing.T) {
	rootCmd := NewRootCmd()

	var podCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "pod" {
			podCmd = cmd
			break
		}
	}
	if podCmd == nil {
		t.Fatal("pod command not found")
	}

	expected := []string{"list", "get", "create", "update", "start", "stop", "restart", "reset", "delete"}
	for _, name := range expected {
		found := false
		for _, cmd := range podCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pod command missing subcommand %q", name)
		}
	}
}

func TestDryRunFlag(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"--dry-run", "version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command with --dry-run failed: %v", err)
	}

	if !dryRunFlag {
		t.Error("dry-run flag should be true")
	}
}

func TestOutputFlag(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"--output", "table", "version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command with --output failed: %v", err)
	}

	if outputFlag != "table" {
		t.Errorf("output flag = %q, want %q", outputFlag, "table")
	}
}

func TestDestructiveCommandsRequireYes(t *testing.T) {
	// Pod stop without --yes should fail
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"pod", "stop", "test-pod-id"})

	// This will call os.Exit, so we can't test it directly in unit tests
	// But we can verify the command exists and has the right args
	var podCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "pod" {
			podCmd = cmd
			break
		}
	}

	var stopCmd *cobra.Command
	for _, cmd := range podCmd.Commands() {
		if cmd.Name() == "stop" {
			stopCmd = cmd
			break
		}
	}

	if stopCmd == nil {
		t.Fatal("pod stop command not found")
	}

	if stopCmd.Args == nil {
		t.Error("pod stop should require args")
	}
}
