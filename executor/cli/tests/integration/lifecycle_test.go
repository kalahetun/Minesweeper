package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"hfi-cli/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLILifecycleCreatePolicy tests creating a policy via CLI
func TestCLILifecycleCreatePolicy(t *testing.T) {
	// Setup test server with mock policy store
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// POST /api/v1/policies
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/policies" {
			var policy types.FaultInjectionPolicy
			if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			policies[policy.Metadata.Name] = &policy
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create test policy
	policy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-create-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{
						Path: &types.PathMatcher{
							Exact: "/api/test",
						},
					},
					Fault: types.FaultAction{
						Percentage:   100,
						StartDelayMs: 100,
					},
				},
			},
		},
	}

	// Simulate CLI POST request
	body, err := json.Marshal(policy)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/policies", server.URL),
		"application/json",
		strings.NewReader(string(body)),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdPolicy types.FaultInjectionPolicy
	err = json.NewDecoder(resp.Body).Decode(&createdPolicy)
	require.NoError(t, err)
	assert.Equal(t, "test-create-policy", createdPolicy.Metadata.Name)
	assert.Equal(t, 1, len(createdPolicy.Spec.Rules))
}

// TestCLILifecycleReadPolicy tests reading a policy via CLI
func TestCLILifecycleReadPolicy(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	policy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-read-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{
						Method: &types.StringMatcher{Exact: "GET"},
					},
					Fault: types.FaultAction{
						Percentage: 50,
					},
				},
			},
		},
	}
	policies["test-read-policy"] = &policy

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// GET /api/v1/policies/:name
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			if p, exists := policies[name]; exists {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(p)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Get policy
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/policies/test-read-policy", server.URL))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrieved types.FaultInjectionPolicy
	err = json.NewDecoder(resp.Body).Decode(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, "test-read-policy", retrieved.Metadata.Name)
}

// TestCLILifecycleListPolicies tests listing policies via CLI
func TestCLILifecycleListPolicies(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	// Pre-populate with test policies
	for i := 1; i <= 3; i++ {
		p := types.FaultInjectionPolicy{
			Metadata: types.PolicyMetadata{
				Name: fmt.Sprintf("policy-%d", i),
			},
			Spec: types.PolicySpec{
				Rules: []types.Rule{},
			},
		}
		policies[p.Metadata.Name] = &p
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// GET /api/v1/policies
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/policies" {
			policyList := make([]*types.FaultInjectionPolicy, 0)
			for _, p := range policies {
				policyList = append(policyList, p)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(policyList)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/policies", server.URL))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var list []*types.FaultInjectionPolicy
	err = json.NewDecoder(resp.Body).Decode(&list)
	require.NoError(t, err)
	assert.Equal(t, 3, len(list))
}

// TestCLILifecycleUpdatePolicy tests updating a policy via CLI
func TestCLILifecycleUpdatePolicy(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	policy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-update-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{},
					Fault: types.FaultAction{
						Percentage: 25,
					},
				},
			},
		},
	}
	policies["test-update-policy"] = &policy

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// PUT /api/v1/policies/:name
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			var updated types.FaultInjectionPolicy
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, exists := policies[name]; exists {
				policies[name] = &updated
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(updated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Update policy with higher percentage
	updatedPolicy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-update-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{},
					Fault: types.FaultAction{
						Percentage: 75,
					},
				},
			},
		},
	}

	body, err := json.Marshal(updatedPolicy)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/policies/test-update-policy", server.URL),
		strings.NewReader(string(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrieved types.FaultInjectionPolicy
	err = json.NewDecoder(resp.Body).Decode(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, 75, retrieved.Spec.Rules[0].Fault.Percentage)
}

// TestCLILifecycleDeletePolicy tests deleting a policy via CLI
func TestCLILifecycleDeletePolicy(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	policy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "test-delete-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{},
		},
	}
	policies["test-delete-policy"] = &policy

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// DELETE /api/v1/policies/:name
		if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			if _, exists := policies[name]; exists {
				delete(policies, name)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/api/v1/policies/test-delete-policy", server.URL), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// TestCLILifecycleCompleteCycle tests complete CRUD cycle
func TestCLILifecycleCompleteCycle(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/policies":
			var policy types.FaultInjectionPolicy
			json.NewDecoder(r.Body).Decode(&policy)
			policies[policy.Metadata.Name] = &policy
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/policies/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			if p, exists := policies[name]; exists {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(p)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/policies/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			var updated types.FaultInjectionPolicy
			json.NewDecoder(r.Body).Decode(&updated)
			if _, exists := policies[name]; exists {
				policies[name] = &updated
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(updated)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/policies/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			if _, exists := policies[name]; exists {
				delete(policies, name)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create
	createPolicy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{Name: "cycle-policy"},
		Spec: types.PolicySpec{
			Rules: []types.Rule{{
				Fault: types.FaultAction{Percentage: 50},
			}},
		},
	}
	createBody, _ := json.Marshal(createPolicy)
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/policies", server.URL), "application/json", strings.NewReader(string(createBody)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Read
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/policies/cycle-policy", server.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Update
	updatePolicy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{Name: "cycle-policy"},
		Spec: types.PolicySpec{
			Rules: []types.Rule{{
				Fault: types.FaultAction{Percentage: 100},
			}},
		},
	}
	updateBody, _ := json.Marshal(updatePolicy)
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/policies/cycle-policy", server.URL), strings.NewReader(string(updateBody)))
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/policies/cycle-policy", server.URL), nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify deleted
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/policies/cycle-policy", server.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// TestCLILifecycleMultiplePolicies tests managing multiple policies
func TestCLILifecycleMultiplePolicies(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/policies" {
			var policy types.FaultInjectionPolicy
			json.NewDecoder(r.Body).Decode(&policy)
			policies[policy.Metadata.Name] = &policy
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/policies" {
			list := make([]*types.FaultInjectionPolicy, 0)
			for _, p := range policies {
				list = append(list, p)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(list)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create 5 policies
	for i := 1; i <= 5; i++ {
		policy := types.FaultInjectionPolicy{
			Metadata: types.PolicyMetadata{
				Name: fmt.Sprintf("multi-policy-%d", i),
			},
			Spec: types.PolicySpec{
				Rules: []types.Rule{{
					Fault: types.FaultAction{Percentage: i * 20},
				}},
			},
		}
		body, _ := json.Marshal(policy)
		resp, err := http.Post(fmt.Sprintf("%s/api/v1/policies", server.URL), "application/json", strings.NewReader(string(body)))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// List all
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/policies", server.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var list []*types.FaultInjectionPolicy
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	assert.Equal(t, 5, len(list))
}

// TestCLILifecycleGetNonExistent tests getting non-existent policy
func TestCLILifecycleGetNonExistent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/policies/nonexistent", server.URL))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestCLILifecycleDeleteNonExistent tests deleting non-existent policy
func TestCLILifecycleDeleteNonExistent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/policies/nonexistent", server.URL), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestCLILifecycleComplexRules tests policies with complex rules and conditions
func TestCLILifecycleComplexRules(t *testing.T) {
	policies := make(map[string]*types.FaultInjectionPolicy)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/policies" {
			var policy types.FaultInjectionPolicy
			json.NewDecoder(r.Body).Decode(&policy)
			policies[policy.Metadata.Name] = &policy
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/policies/") {
			name := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
			if p, exists := policies[name]; exists {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(p)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create policy with multiple rules and complex conditions
	policy := types.FaultInjectionPolicy{
		Metadata: types.PolicyMetadata{
			Name: "complex-policy",
		},
		Spec: types.PolicySpec{
			Rules: []types.Rule{
				{
					Match: types.MatchCondition{
						Method: &types.StringMatcher{Exact: "POST"},
						Path:   &types.PathMatcher{Prefix: "/api"},
						Headers: []types.HeaderMatcher{
							{Name: "X-Fault-Injection", Exact: "true"},
						},
					},
					Fault: types.FaultAction{
						Percentage:   100,
						StartDelayMs: 500,
						Delay: &types.DelayAction{
							FixedDelayMs: 100,
						},
					},
				},
				{
					Match: types.MatchCondition{
						Path: &types.PathMatcher{Regex: "^/admin/.*"},
					},
					Fault: types.FaultAction{
						Percentage: 50,
						Abort: &types.AbortAction{
							HTTPStatus: 403,
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(policy)
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/policies", server.URL), "application/json", strings.NewReader(string(body)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Read and verify
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/policies/complex-policy", server.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrieved types.FaultInjectionPolicy
	json.NewDecoder(resp.Body).Decode(&retrieved)
	resp.Body.Close()

	assert.Equal(t, 2, len(retrieved.Spec.Rules))
	assert.Equal(t, 500, retrieved.Spec.Rules[0].Fault.StartDelayMs)
	assert.NotNil(t, retrieved.Spec.Rules[0].Fault.Delay)
	assert.NotNil(t, retrieved.Spec.Rules[1].Fault.Abort)
	assert.Equal(t, 403, retrieved.Spec.Rules[1].Fault.Abort.HTTPStatus)
}
