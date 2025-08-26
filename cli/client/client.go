package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"hfi-cli/types"
)

// GlobalFlags holds global CLI configuration
type GlobalFlags struct {
	ControlPlaneAddr string
	Timeout          time.Duration
}

// APIError represents an error from the Control Plane API
type APIError struct {
	StatusCode int
	ErrCode    string
	Message    string
}

func (e *APIError) Error() string {
	if e.ErrCode != "" {
		return fmt.Sprintf("failed with status %d (%s): %s", e.StatusCode, e.ErrCode, e.Message)
	}
	return fmt.Sprintf("failed with status %d: %s", e.StatusCode, e.Message)
}

// IAPIClient defines the interface for interacting with the Control Plane API
type IAPIClient interface {
	// Policy operations
	CreateOrUpdatePolicy(ctx context.Context, policy *types.FaultInjectionPolicy) error
	GetPolicyByName(ctx context.Context, name string) (*types.FaultInjectionPolicy, error)
	ListPolicies(ctx context.Context) ([]*types.FaultInjectionPolicy, error)
	DeletePolicy(ctx context.Context, name string) error
	
	// Health check
	HealthCheck(ctx context.Context) error
}

// APIClient is the concrete implementation of IAPIClient
type APIClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewAPIClient creates a new API client instance
func NewAPIClient(baseURL string, timeout time.Duration) (*APIClient, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid control plane address: %w", err)
	}
	
	// Validate URL scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s (expected http or https)", parsedURL.Scheme)
	}
	
	httpClient := &http.Client{
		Timeout: timeout,
	}
	
	return &APIClient{
		baseURL:    parsedURL,
		httpClient: httpClient,
	}, nil
}

// HealthCheck verifies that the Control Plane API is reachable
func (c *APIClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("control plane health check failed: %s", http.StatusText(resp.StatusCode)),
		}
	}
	
	return nil
}

// CreateOrUpdatePolicy creates or updates a fault injection policy
func (c *APIClient) CreateOrUpdatePolicy(ctx context.Context, policy *types.FaultInjectionPolicy) error {
	jsonData, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	url := fmt.Sprintf("%s/v1/policies", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	return nil
}

// GetPolicyByName retrieves a specific policy by name
func (c *APIClient) GetPolicyByName(ctx context.Context, name string) (*types.FaultInjectionPolicy, error) {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			ErrCode:    "NotFound",
			Message:    fmt.Sprintf("policy %q not found", name),
		}
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var policy types.FaultInjectionPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &policy, nil
}

// ListPolicies retrieves all policies
func (c *APIClient) ListPolicies(ctx context.Context) ([]*types.FaultInjectionPolicy, error) {
	url := fmt.Sprintf("%s/v1/policies", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	// Control Plane returns {"policies": [...]}, so we need a wrapper struct
	var response struct {
		Policies []*types.FaultInjectionPolicy `json:"policies"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Policies, nil
}

// DeletePolicy deletes a policy by name
func (c *APIClient) DeletePolicy(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &APIError{
			StatusCode: resp.StatusCode,
			ErrCode:    "NotFound",
			Message:    fmt.Sprintf("policy %q not found", name),
		}
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	return nil
}
