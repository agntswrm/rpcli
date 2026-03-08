package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kp, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	if kp.PrivateKeyPath == "" || kp.PublicKeyPath == "" {
		t.Error("key paths should not be empty")
	}

	if !strings.HasPrefix(kp.PublicKey, "ssh-rsa ") {
		t.Errorf("public key should start with 'ssh-rsa ', got: %s", kp.PublicKey[:20])
	}

	// Verify files exist
	if _, err := os.Stat(kp.PrivateKeyPath); os.IsNotExist(err) {
		t.Error("private key file should exist")
	}
	if _, err := os.Stat(kp.PublicKeyPath); os.IsNotExist(err) {
		t.Error("public key file should exist")
	}

	// Verify private key permissions
	info, err := os.Stat(kp.PrivateKeyPath)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("private key perms = %o, want 0600", info.Mode().Perm())
	}
}

func TestGetLocalKey(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// No key yet
	kp, err := GetLocalKey()
	if err != nil {
		t.Fatalf("GetLocalKey() error: %v", err)
	}
	if kp != nil {
		t.Error("should return nil when no key exists")
	}

	// Generate one, then retrieve
	generated, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	kp, err = GetLocalKey()
	if err != nil {
		t.Fatalf("GetLocalKey() error: %v", err)
	}
	if kp == nil {
		t.Fatal("should return key pair after generation")
	}
	if kp.PublicKey != generated.PublicKey {
		t.Error("retrieved key should match generated key")
	}
}

func TestSSHDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	dir, err := SSHDir()
	if err != nil {
		t.Fatalf("SSHDir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".rpcli", "ssh")
	if dir != expected {
		t.Errorf("SSHDir() = %q, want %q", dir, expected)
	}
}
