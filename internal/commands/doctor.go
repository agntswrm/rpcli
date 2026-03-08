package commands

import (
	"fmt"

	"github.com/agntswrm/rpcli/internal/config"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/agntswrm/rpcli/internal/ssh"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check and fix rpcli setup (API key, SSH keys, account config)",
		Long:  "Runs diagnostics and auto-fixes common setup issues: verifies API key, generates SSH keys if missing, and registers them with your Runpod account.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			type checkResult struct {
				Check   string `json:"check"`
				Status  string `json:"status"`
				Message string `json:"message,omitempty"`
				Action  string `json:"action,omitempty"`
			}

			var results []checkResult

			// 1. Check API key
			apiKey := config.ResolveAPIKey(apiKeyFlag)
			if apiKey == "" {
				results = append(results, checkResult{
					Check:   "api_key",
					Status:  "fail",
					Message: "No API key configured",
					Action:  "Run 'rpcli config set-key <key>' or set RUNPOD_API_KEY env var",
				})
				return output.Print(getFormat(), map[string]any{
					"status": "incomplete",
					"checks": results,
				})
			}
			results = append(results, checkResult{
				Check:  "api_key",
				Status: "ok",
			})

			// 2. Verify API key works
			client := getClient()
			query := `query { myself { id pubKey } }`
			var accountResult struct {
				Myself struct {
					ID     string `json:"id"`
					PubKey string `json:"pubKey"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &accountResult); err != nil {
				results = append(results, checkResult{
					Check:   "api_connection",
					Status:  "fail",
					Message: fmt.Sprintf("API connection failed: %v", err),
					Action:  "Check your API key and network connection",
				})
				return output.Print(getFormat(), map[string]any{
					"status": "incomplete",
					"checks": results,
				})
			}
			results = append(results, checkResult{
				Check:  "api_connection",
				Status: "ok",
			})

			// 3. Check for local SSH key
			localKey, err := ssh.GetLocalKey()
			if err != nil {
				results = append(results, checkResult{
					Check:   "ssh_key_local",
					Status:  "fail",
					Message: fmt.Sprintf("Error reading local key: %v", err),
				})
			} else if localKey == nil {
				// No local key — generate one
				if dryRunFlag {
					results = append(results, checkResult{
						Check:  "ssh_key_local",
						Status: "missing",
						Action: "Would generate SSH key pair",
					})
				} else {
					kp, err := ssh.GenerateKey()
					if err != nil {
						results = append(results, checkResult{
							Check:   "ssh_key_local",
							Status:  "fail",
							Message: fmt.Sprintf("Failed to generate key: %v", err),
						})
					} else {
						localKey = kp
						results = append(results, checkResult{
							Check:  "ssh_key_local",
							Status: "fixed",
							Action: fmt.Sprintf("Generated new SSH key at %s", kp.PrivateKeyPath),
						})
					}
				}
			} else {
				results = append(results, checkResult{
					Check:  "ssh_key_local",
					Status: "ok",
				})
			}

			// 4. Check if SSH key is registered with account
			if localKey != nil {
				keyRegistered := false
				if accountResult.Myself.PubKey != "" {
					keyRegistered = containsKey(accountResult.Myself.PubKey, localKey.PublicKey)
				}

				if !keyRegistered {
					if dryRunFlag {
						results = append(results, checkResult{
							Check:  "ssh_key_account",
							Status: "missing",
							Action: "Would add local SSH key to Runpod account",
						})
					} else {
						// Add key to account
						newKeys := accountResult.Myself.PubKey
						if newKeys != "" {
							newKeys += "\n"
						}
						newKeys += localKey.PublicKey

						mutation := `mutation($input: UpdateUserSettingsInput) {
							updateUserSettings(input: $input) { id }
						}`
						vars := map[string]any{
							"input": map[string]any{"pubKey": newKeys},
						}
						if err := client.Execute(mutation, vars, nil); err != nil {
							results = append(results, checkResult{
								Check:   "ssh_key_account",
								Status:  "fail",
								Message: fmt.Sprintf("Failed to register key: %v", err),
							})
						} else {
							results = append(results, checkResult{
								Check:  "ssh_key_account",
								Status: "fixed",
								Action: "SSH key added to Runpod account",
							})
						}
					}
				} else {
					results = append(results, checkResult{
						Check:  "ssh_key_account",
						Status: "ok",
					})
				}
			}

			// Determine overall status
			overallStatus := "ok"
			for _, r := range results {
				if r.Status == "fail" {
					overallStatus = "fail"
					break
				}
				if r.Status == "fixed" || r.Status == "missing" {
					overallStatus = "fixed"
				}
			}

			return output.Print(getFormat(), map[string]any{
				"status": overallStatus,
				"checks": results,
			})
		},
	}
}

// containsKey checks if pubKeyBlob contains the given key (ignoring comments).
func containsKey(pubKeyBlob, key string) bool {
	// Extract just the key type + key data (without comment) for comparison
	keyParts := splitKeyParts(key)
	for _, line := range splitLines(pubKeyBlob) {
		lineParts := splitKeyParts(line)
		if len(lineParts) >= 2 && len(keyParts) >= 2 {
			if lineParts[0] == keyParts[0] && lineParts[1] == keyParts[1] {
				return true
			}
		}
	}
	return false
}

func splitKeyParts(key string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == ' ' {
			if i > start {
				parts = append(parts, key[start:i])
			}
			start = i + 1
		}
	}
	if start < len(key) {
		parts = append(parts, key[start:])
	}
	return parts
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
