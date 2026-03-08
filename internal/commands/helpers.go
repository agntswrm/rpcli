package commands

import "strings"

// parseEnvVars parses KEY=VALUE strings into maps.
func parseEnvVars(envVars []string) []map[string]string {
	envList := make([]map[string]string, 0, len(envVars))
	for _, e := range envVars {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envList = append(envList, map[string]string{"key": parts[0], "value": parts[1]})
		}
	}
	return envList
}
