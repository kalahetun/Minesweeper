package storage

import (
	"testing"
)

func TestEtcdStoreCreateUpdate(t *testing.T) {
	// Skip test if no etcd instance is available
	t.Skip("Requires running etcd instance")

	// This test would require a running etcd instance
	store, err := NewEtcdStore([]string{"localhost:2379"})
	if err != nil {
		t.Skipf("Skipping etcd test: %v", err)
	}
	defer store.Close()

	policy := &FaultInjectionPolicy{
		Metadata: Metadata{
			Name:    "test-policy",
			Version: "1.0.0",
		},
		Spec: PolicySpec{
			Rules: []Rule{
				{
					Match: MatchCondition{
						Path: &PathMatcher{Exact: "/test"},
					},
					Fault: FaultAction{
						Percentage: 50,
						Delay: &DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	// Test Create
	err = store.Create(policy)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// Test Create duplicate should fail
	err = store.Create(policy)
	if err != ErrAlreadyExists {
		t.Errorf("Expected ErrAlreadyExists, got: %v", err)
	}

	// Test Update
	policy.Metadata.Version = "1.0.1"
	err = store.Update(policy)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Test Update non-existent policy
	nonExistentPolicy := &FaultInjectionPolicy{
		Metadata: Metadata{
			Name:    "non-existent",
			Version: "1.0.0",
		},
		Spec: PolicySpec{
			Rules: []Rule{
				{
					Match: MatchCondition{
						Path: &PathMatcher{Exact: "/test"},
					},
					Fault: FaultAction{
						Percentage: 50,
						Delay: &DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	err = store.Update(nonExistentPolicy)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}

	// Clean up
	store.Delete("test-policy")
}

func TestMemoryStoreCreateUpdate(t *testing.T) {
	store := NewMemoryStore()

	policy := &FaultInjectionPolicy{
		Metadata: Metadata{
			Name:    "test-policy",
			Version: "1.0.0",
		},
		Spec: PolicySpec{
			Rules: []Rule{
				{
					Match: MatchCondition{
						Path: &PathMatcher{Exact: "/test"},
					},
					Fault: FaultAction{
						Percentage: 50,
						Delay: &DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	// Test Create
	err := store.Create(policy)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// Test Create duplicate should fail
	err = store.Create(policy)
	if err != ErrAlreadyExists {
		t.Errorf("Expected ErrAlreadyExists, got: %v", err)
	}

	// Test Update
	policy.Metadata.Version = "1.0.1"
	err = store.Update(policy)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Test Update non-existent policy
	nonExistentPolicy := &FaultInjectionPolicy{
		Metadata: Metadata{
			Name:    "non-existent",
			Version: "1.0.0",
		},
		Spec: PolicySpec{
			Rules: []Rule{
				{
					Match: MatchCondition{
						Path: &PathMatcher{Exact: "/test"},
					},
					Fault: FaultAction{
						Percentage: 50,
						Delay: &DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	err = store.Update(nonExistentPolicy)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}

	// Verify the policy exists and has the correct version
	retrievedPolicy, err := store.Get("test-policy")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if retrievedPolicy.Metadata.Version != "1.0.1" {
		t.Errorf("Expected version 1.0.1, got: %s", retrievedPolicy.Metadata.Version)
	}
}

func TestInvalidInputErrors(t *testing.T) {
	store := NewMemoryStore()

	// Test Create with nil policy
	err := store.Create(nil)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for nil policy, got: %v", err)
	}

	// Test Create with empty name
	policy := &FaultInjectionPolicy{
		Metadata: Metadata{
			Name:    "",
			Version: "1.0.0",
		},
	}

	err = store.Create(policy)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for empty name, got: %v", err)
	}

	// Test Update with nil policy
	err = store.Update(nil)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for nil policy, got: %v", err)
	}

	// Test Update with empty name
	err = store.Update(policy)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for empty name, got: %v", err)
	}
}
