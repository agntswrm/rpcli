package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "*****"},
		{"abcdefghij", "abcd**efgh"}, // wait, len=10, 10-8=2 stars... let me recalc
		// len=10: first 4 + 2 stars + last 4 = "abcd**ghij"
	}

	// Fix expected values based on actual logic
	tests = []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "*****"},                      // len 5 <= 8
		{"12345678", "********"},                 // len 8 <= 8
		{"123456789", "1234*6789"},               // len 9: first4 + 1 star + last4
		{"1234567890ab", "1234****90ab"},          // len 12: first4 + 4 stars + last4
	}

	for _, tt := range tests {
		result := MaskKey(tt.input)
		if result != tt.expected {
			t.Errorf("MaskKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestResolveAPIKey(t *testing.T) {
	// Flag takes priority
	result := ResolveAPIKey("flag-key")
	if result != "flag-key" {
		t.Errorf("ResolveAPIKey with flag = %q, want %q", result, "flag-key")
	}

	// Env takes priority over config
	os.Setenv("RUNPOD_API_KEY", "env-key")
	defer os.Unsetenv("RUNPOD_API_KEY")

	result = ResolveAPIKey("")
	if result != "env-key" {
		t.Errorf("ResolveAPIKey with env = %q, want %q", result, "env-key")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp dir
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Ensure config dir exists
	os.MkdirAll(filepath.Join(tmpDir, ".config", "rpcli"), 0700)

	cfg := &Config{APIKey: "test-key-12345"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.APIKey != cfg.APIKey {
		t.Errorf("Load().APIKey = %q, want %q", loaded.APIKey, cfg.APIKey)
	}
}
