package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.runpod.io/graphql"

// Client is a GraphQL client for the Runpod API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	version    string
}

// NewClient creates a new Runpod API client.
func NewClient(apiKey, version string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		version: version,
	}
}

// graphQLRequest is the request body for a GraphQL query.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the response body from a GraphQL query.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

// Execute runs a GraphQL query and unmarshals the result.
func (c *Client) Execute(query string, variables map[string]any, result any) error {
	if c.apiKey == "" {
		return fmt.Errorf("API key not configured. Run 'rpcli config set-key' or set RUNPOD_API_KEY")
	}

	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?api_key=%s", c.baseURL, c.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.version != "" {
		req.Header.Set("User-Agent", "rpcli/"+c.version)
	} else {
		req.Header.Set("User-Agent", "rpcli/dev")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: invalid or expired API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// If we have data, unmarshal it even if there are partial errors (common in GraphQL).
	// Only fail on errors if there is no usable data.
	if result != nil && len(gqlResp.Data) > 0 && string(gqlResp.Data) != "null" {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("failed to parse data: %w", err)
		}
		return nil
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("API error: %s", gqlResp.Errors[0].Message)
	}

	return nil
}
