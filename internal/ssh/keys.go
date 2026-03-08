package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

const (
	defaultKeyName = "rpcli-key"
	keyBits        = 2048
)

// KeyPair holds paths to an SSH key pair.
type KeyPair struct {
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
	PublicKey      string `json:"public_key"`
}

// SSHDir returns the path to the rpcli SSH directory.
func SSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".rpcli", "ssh"), nil
}

// GetLocalKey reads the existing local SSH key pair if it exists.
func GetLocalKey() (*KeyPair, error) {
	dir, err := SSHDir()
	if err != nil {
		return nil, err
	}

	privPath := filepath.Join(dir, defaultKeyName)
	pubPath := filepath.Join(dir, defaultKeyName+".pub")

	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		return nil, nil // no key exists
	}

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	return &KeyPair{
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		PublicKey:      strings.TrimSpace(string(pubBytes)),
	}, nil
}

// GenerateKey generates a new RSA SSH key pair and saves it to disk.
func GenerateKey() (*KeyPair, error) {
	dir, err := SSHDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Encode private key as PEM
	privBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Derive SSH public key
	pub, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}
	pubBytes := gossh.MarshalAuthorizedKey(pub)
	pubStr := strings.TrimSpace(string(pubBytes)) + " " + defaultKeyName

	// Write files
	privPath := filepath.Join(dir, defaultKeyName)
	pubPath := filepath.Join(dir, defaultKeyName+".pub")

	if err := os.WriteFile(privPath, privBytes, 0600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(pubStr+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %w", err)
	}

	return &KeyPair{
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		PublicKey:      pubStr,
	}, nil
}
