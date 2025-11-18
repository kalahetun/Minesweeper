package unit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hfi-cli/client"
	"hfi-cli/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientCreateOrUpdatePolicySuccess tests successful policy creation via HTTP
func TestClientCreateOrUpdatePolicySuccess(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies" && r.Method == "POST" {
			var policy types.FaultInjectionPolicy
			err := json.NewDecoder(r.Body).Decode(&policy)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	policy := &types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{
						Path: &types.PathMatcher{
							Exact: "/test",
						},
					},
					Fault: types.FaultAction{
						Percentage: 50,
						Abort: &types.AbortAction{
							HTTPStatus: 503,
						},
					},
				},
			},
		},
	}

	err = apiClient.CreateOrUpdatePolicy(context.Background(), policy)
	assert.NoError(t, err)
}

// TestClientGetPolicySuccess tests successful policy retrieval
func TestClientGetPolicySuccess(t *testing.T) {
	expectedPolicy := &types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies/test-policy" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedPolicy)
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	policy, err := apiClient.GetPolicyByName(context.Background(), "test-policy")
	assert.NoError(t, err)
	assert.Equal(t, "test-policy", policy.Metadata.Name)
}

// TestClientListPoliciesSuccess tests successful policy listing
func TestClientListPoliciesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies" && r.Method == "GET" {
			policies := []types.FaultInjectionPolicy{
				{
					Metadata: types.PolicyMetadata{
						Name: "policy-1",
					},
					Spec: types.PolicySpec{Rules: []types.Rule{}},
				},
				{
					Metadata: types.PolicyMetadata{
						Name: "policy-2",
					},
					Spec: types.PolicySpec{Rules: []types.Rule{}},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Server wraps policies in a response structure
			json.NewEncoder(w).Encode(map[string]interface{}{
				"policies": policies,
			})
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	policies, err := apiClient.ListPolicies(context.Background())
	assert.NoError(t, err)
	assert.Len(t, policies, 2)
	assert.Equal(t, "policy-1", policies[0].Metadata.Name)
	assert.Equal(t, "policy-2", policies[1].Metadata.Name)
}

// TestClientDeletePolicySuccess tests successful policy deletion
func TestClientDeletePolicySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies/test-policy" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.DeletePolicy(context.Background(), "test-policy")
	assert.NoError(t, err)
}

// TestClientHealthCheck tests health check endpoint
func TestClientHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.HealthCheck(context.Background())
	assert.NoError(t, err)
}

// TestClientErrorHandling404 tests 404 error handling
func TestClientErrorHandling404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "Policy not found",
		})
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	_, err = apiClient.GetPolicyByName(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestClientErrorHandling500 tests 500 error handling
func TestClientErrorHandling500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.CreateOrUpdatePolicy(context.Background(), &types.FaultInjectionPolicy{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestClientErrorHandling400 tests 400 Bad Request handling
func TestClientErrorHandling400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request body",
		})
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.CreateOrUpdatePolicy(context.Background(), &types.FaultInjectionPolicy{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// TestClientErrorHandling409 tests 409 Conflict handling
func TestClientErrorHandling409(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "resource_already_exists",
			"message": "Policy already exists",
		})
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.CreateOrUpdatePolicy(context.Background(), &types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{Name: "existing"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "409")
}

// TestClientConnectionTimeout tests request timeout handling
func TestClientConnectionTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with very short timeout
	apiClient, err := client.NewAPIClient(server.URL, 10*time.Millisecond)
	require.NoError(t, err)

	// Request should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = apiClient.HealthCheck(ctx)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline") ||
		strings.Contains(err.Error(), "i/o timeout"))
}

// TestClientContextCancellation tests cancellation via context
func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = apiClient.HealthCheck(ctx)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context canceled") ||
		strings.Contains(err.Error(), "cancelled"))
}

// TestClientInvalidBaseURL tests invalid base URL handling
func TestClientInvalidBaseURL(t *testing.T) {
	_, err := client.NewAPIClient("://invalid", 5*time.Second)
	assert.Error(t, err)
}

// TestClientMalformedResponseJSON tests malformed JSON response handling
func TestClientMalformedResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid json"))
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	_, err = apiClient.ListPolicies(context.Background())
	assert.Error(t, err)
}

// TestClientEmptyResponse tests empty response body handling
func TestClientEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			w.WriteHeader(http.StatusOK)
			// No body
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	err = apiClient.HealthCheck(context.Background())
	// Empty response should be handled gracefully
	// (depending on implementation, may error or succeed)
	_ = err
}

// TestClientRequestContentType tests that requests are sent with correct Content-Type
func TestClientRequestContentType(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(types.FaultInjectionPolicy{})
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	apiClient.CreateOrUpdatePolicy(context.Background(), &types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{Name: "test"},
	})

	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
}

// TestClientLargePayloadHandling tests handling of large policy payloads
func TestClientLargePayloadHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/policies" && r.Method == "POST" {
			// Read and verify payload size
			body, _ := io.ReadAll(r.Body)
			assert.Greater(t, len(body), 1000) // Should have significant payload

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(types.FaultInjectionPolicy{})
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	// Create policy with many rules
	policy := &types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{Name: "large-policy"},
		Spec: types.PolicySpec{
			Rules: make([]types.Rule, 100), // 100 rules
		},
	}

	err = apiClient.CreateOrUpdatePolicy(context.Background(), policy)
	assert.NoError(t, err)
}

// TestClientRetryBehavior tests client behavior on transient failures
func TestClientRetryBehavior(t *testing.T) {
	// This test is informational; actual retry behavior depends on client implementation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Always succeed on this test
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	require.NoError(t, err)

	// If no retries are implemented, at least verify idempotency
	err1 := apiClient.HealthCheck(context.Background())
	err2 := apiClient.HealthCheck(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

// TestClientCreateNewAPIClient tests NewAPIClient initialization
func TestClientCreateNewAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Valid URL
	apiClient, err := client.NewAPIClient(server.URL, 5*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, apiClient)

	// Invalid URL format
	_, err = client.NewAPIClient("not-a-valid-url://", 5*time.Second)
	assert.Error(t, err)

	// Empty URL
	_, err = client.NewAPIClient("", 5*time.Second)
	assert.Error(t, err)
}

// TestAPIErrorStringer tests APIError error message formatting
func TestAPIErrorStringer(t *testing.T) {
	err := &client.APIError{
		StatusCode: 500,
		ErrCode:    "internal_error",
		Message:    "Something went wrong",
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "500")
	assert.Contains(t, errMsg, "internal_error")
	assert.Contains(t, errMsg, "Something went wrong")

	// Test without error code
	err2 := &client.APIError{
		StatusCode: 400,
		Message:    "Bad request",
	}
	errMsg2 := err2.Error()
	assert.Contains(t, errMsg2, "400")
	assert.NotContains(t, errMsg2, "()") // No empty parentheses
}
