package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/agntswrm/rpcli/internal/api"
	"github.com/agntswrm/rpcli/internal/output"
	"github.com/agntswrm/rpcli/internal/ssh"
	"github.com/spf13/cobra"
)

func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh",
		Short: "Manage SSH keys and connections to pods",
	}

	cmd.AddCommand(newSSHListKeysCmd())
	cmd.AddCommand(newSSHAddKeyCmd())
	cmd.AddCommand(newSSHInfoCmd())

	return cmd
}

func newSSHListKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-keys",
		Short: "List SSH keys on your Runpod account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query { myself { pubKey } }`

			var result struct {
				Myself struct {
					PubKey string `json:"pubKey"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			keys := parseAuthorizedKeys(result.Myself.PubKey)
			return output.Print(getFormat(), keys)
		},
	}
}

func newSSHAddKeyCmd() *cobra.Command {
	var (
		key     string
		keyFile string
	)

	cmd := &cobra.Command{
		Use:   "add-key",
		Short: "Add an SSH public key to your Runpod account",
		Long:  "Add an SSH public key. If no key is provided, generates a new key pair automatically.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pubKey string

			if key != "" {
				pubKey = strings.TrimSpace(key)
			} else if keyFile != "" {
				data, err := os.ReadFile(keyFile)
				if err != nil {
					exitError("file_error", fmt.Sprintf("Failed to read key file: %v", err))
				}
				pubKey = strings.TrimSpace(string(data))
			} else {
				// Auto-generate a key
				existing, err := ssh.GetLocalKey()
				if err != nil {
					exitError("ssh_error", err.Error())
				}
				if existing != nil {
					pubKey = existing.PublicKey
				} else {
					kp, err := ssh.GenerateKey()
					if err != nil {
						exitError("ssh_error", err.Error())
					}
					pubKey = kp.PublicKey
				}
			}

			if dryRunFlag {
				return output.Print(getFormat(), map[string]any{
					"dry_run":    true,
					"action":     "ssh_add_key",
					"public_key": pubKey,
				})
			}

			client := getClient()

			// Get existing keys
			query := `query { myself { pubKey } }`
			var result struct {
				Myself struct {
					PubKey string `json:"pubKey"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			// Check for duplicates
			existingKeys := result.Myself.PubKey
			if strings.Contains(existingKeys, strings.TrimSpace(pubKey)) {
				return output.Print(getFormat(), map[string]string{
					"status":  "ok",
					"message": "SSH key already registered",
				})
			}

			// Append new key
			newKeys := existingKeys
			if newKeys != "" && !strings.HasSuffix(newKeys, "\n") {
				newKeys += "\n"
			}
			newKeys += pubKey

			// Update via API
			mutation := `mutation($input: UpdateUserSettingsInput) {
				updateUserSettings(input: $input) { id }
			}`
			vars := map[string]any{
				"input": map[string]any{
					"pubKey": newKeys,
				},
			}
			if err := client.Execute(mutation, vars, nil); err != nil {
				exitError("api_error", err.Error())
			}

			return output.Print(getFormat(), map[string]string{
				"status":  "ok",
				"message": "SSH key added to account",
			})
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "SSH public key string")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to SSH public key file")

	return cmd
}

func newSSHInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <pod-id>",
		Short: "Show SSH connection command for a pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			query := `query {
				myself {
					pods {
						id name desiredStatus
						runtime {
							ports {
								ip isIpPublic privatePort publicPort type
							}
						}
					}
				}
			}`

			var result struct {
				Myself struct {
					Pods []struct {
						ID            string `json:"id"`
						Name          string `json:"name"`
						DesiredStatus string `json:"desiredStatus"`
						Runtime       *struct {
							Ports []api.PortMapping `json:"ports"`
						} `json:"runtime"`
					} `json:"pods"`
				} `json:"myself"`
			}
			if err := client.Execute(query, nil, &result); err != nil {
				exitError("api_error", err.Error())
			}

			// Find the requested pod
			for _, pod := range result.Myself.Pods {
				if pod.ID != args[0] {
					continue
				}

				if pod.Runtime == nil {
					exitError("pod_not_running", fmt.Sprintf("Pod %s is not running (status: %s)", pod.ID, pod.DesiredStatus))
				}

				// Find SSH port (private port 22)
				for _, port := range pod.Runtime.Ports {
					if port.PrivatePort == 22 && port.IsIPPublic {
						sshCmd := fmt.Sprintf("ssh root@%s -p %d", port.IP, port.PublicPort)

						// Add key path if local key exists
						localKey, _ := ssh.GetLocalKey()
						if localKey != nil {
							sshCmd = fmt.Sprintf("ssh -i %s root@%s -p %d", localKey.PrivateKeyPath, port.IP, port.PublicPort)
						}

						return output.Print(getFormat(), map[string]any{
							"pod_id":  pod.ID,
							"name":    pod.Name,
							"host":    port.IP,
							"port":    port.PublicPort,
							"user":    "root",
							"command": sshCmd,
						})
					}
				}

				exitError("no_ssh_port", fmt.Sprintf("Pod %s has no public SSH port. Make sure port 22/tcp is exposed.", pod.ID))
				return nil
			}

			exitError("not_found", fmt.Sprintf("Pod %s not found", args[0]))
			return nil
		},
	}
}

type sshKeyInfo struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
}

func parseAuthorizedKeys(pubKeyBlob string) []sshKeyInfo {
	var keys []sshKeyInfo
	for _, line := range strings.Split(pubKeyBlob, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		entry := sshKeyInfo{}
		if len(parts) >= 1 {
			entry.Type = parts[0]
		}
		if len(parts) >= 2 {
			// Truncate the key for display
			k := parts[1]
			if len(k) > 20 {
				k = k[:10] + "..." + k[len(k)-10:]
			}
			entry.Fingerprint = k
		}
		if len(parts) >= 3 {
			entry.Comment = strings.Join(parts[2:], " ")
		}
		keys = append(keys, entry)
	}
	return keys
}
