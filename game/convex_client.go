package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ConvexClient provides access to Convex database via HTTP API
type ConvexClient struct {
	baseURL    string
	httpClient *http.Client
}

// AIScript represents an AI script from Convex
type AIScript struct {
	ID          string  `json:"_id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description,omitempty"`
	CreatedAt   int64   `json:"createdAt"`
	UpdatedAt   *int64  `json:"updatedAt,omitempty"`
}

// convexQueryRequest represents the request body for Convex HTTP API
type convexQueryRequest struct {
	Path   string         `json:"path"`
	Args   map[string]any `json:"args"`
	Format string         `json:"format"`
}

// convexResponse represents the response from Convex HTTP API
type convexResponse struct {
	Status string          `json:"status"`
	Value  json.RawMessage `json:"value"`
	Error  *string         `json:"errorMessage,omitempty"`
}

// NewConvexClient creates a new Convex HTTP client
func NewConvexClient(deploymentURL string) *ConvexClient {
	return &ConvexClient{
		baseURL: deploymentURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Query executes a Convex query function
func (c *ConvexClient) Query(functionPath string, args map[string]any) (json.RawMessage, error) {
	reqBody := convexQueryRequest{
		Path:   functionPath,
		Args:   args,
		Format: "json",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/query", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("convex API error (status %d): %s", resp.StatusCode, string(body))
	}

	var convexResp convexResponse
	if err := json.Unmarshal(body, &convexResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if convexResp.Status != "success" {
		errMsg := "unknown error"
		if convexResp.Error != nil {
			errMsg = *convexResp.Error
		}
		return nil, fmt.Errorf("convex query failed: %s", errMsg)
	}

	return convexResp.Value, nil
}

// ListAIScripts fetches all AI scripts from Convex
func (c *ConvexClient) ListAIScripts() ([]AIScript, error) {
	result, err := c.Query("aiScripts:list", map[string]any{})
	if err != nil {
		return nil, err
	}

	var scripts []AIScript
	if err := json.Unmarshal(result, &scripts); err != nil {
		return nil, fmt.Errorf("failed to parse AI scripts: %w", err)
	}

	return scripts, nil
}

// GetAIScriptByName fetches a single AI script by name
func (c *ConvexClient) GetAIScriptByName(name string) (*AIScript, error) {
	result, err := c.Query("aiScripts:getByName", map[string]any{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	// Check for null response (script not found)
	if string(result) == "null" {
		return nil, fmt.Errorf("AI script '%s' not found", name)
	}

	var script AIScript
	if err := json.Unmarshal(result, &script); err != nil {
		return nil, fmt.Errorf("failed to parse AI script: %w", err)
	}

	return &script, nil
}

// FetchAIScript fetches the code for an AI script by name
func (c *ConvexClient) FetchAIScript(name string) (string, error) {
	script, err := c.GetAIScriptByName(name)
	if err != nil {
		return "", err
	}
	return script.Code, nil
}
