package service

import (
	"errors"
	"hfi/control-plane/storage"
	"testing"
)

func TestPolicyServiceValidation(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewPolicyService(store)

	// Test policy with invalid name (too short)
	policy := &storage.FaultInjectionPolicy{
		Metadata: storage.Metadata{
			Name:    "ab", // Too short
			Version: "1.0.0",
		},
		Spec: storage.PolicySpec{
			Rules: []storage.Rule{
				{
					Match: storage.MatchCondition{
						Path: &storage.PathMatcher{Exact: "/test"},
					},
					Fault: storage.FaultAction{
						Percentage: 50,
						Delay: &storage.DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	err := service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for short name, got: %v", err)
	}

	// Test policy with invalid name (too long)
	policy.Metadata.Name = "this-is-a-very-long-policy-name-that-exceeds-the-maximum-allowed-length-for-policy-names"
	err = service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for long name, got: %v", err)
	}

	// Test policy with invalid characters in name
	policy.Metadata.Name = "policy@name"
	err = service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for invalid characters, got: %v", err)
	}

	// Test policy with name starting with hyphen
	policy.Metadata.Name = "-invalid-name"
	err = service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for name starting with hyphen, got: %v", err)
	}

	// Test policy with invalid version format
	policy.Metadata.Name = "valid-name"
	policy.Metadata.Version = "1.0" // Invalid format
	err = service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for invalid version format, got: %v", err)
	}

	// Test policy with non-numeric version parts
	policy.Metadata.Version = "1.0.a"
	err = service.CreatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for non-numeric version, got: %v", err)
	}

	// Test valid policy
	policy.Metadata.Name = "valid-policy-name"
	policy.Metadata.Version = "1.0.0"
	err = service.CreatePolicy(policy)
	if err != nil {
		t.Errorf("Expected successful creation for valid policy, got: %v", err)
	}

	// Test creating duplicate policy
	err = service.CreatePolicy(policy)
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Errorf("Expected ErrAlreadyExists for duplicate policy, got: %v", err)
	}
}

func TestPolicyServiceUpdateValidation(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewPolicyService(store)

	// Create initial policy
	policy := &storage.FaultInjectionPolicy{
		Metadata: storage.Metadata{
			Name:    "test-policy",
			Version: "1.0.0",
		},
		Spec: storage.PolicySpec{
			Rules: []storage.Rule{
				{
					Match: storage.MatchCondition{
						Path: &storage.PathMatcher{Exact: "/test"},
					},
					Fault: storage.FaultAction{
						Percentage: 50,
						Delay: &storage.DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	err := service.CreatePolicy(policy)
	if err != nil {
		t.Fatalf("Failed to create initial policy: %v", err)
	}

	// Test updating with same version (should fail)
	err = service.UpdatePolicy(policy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for same version update, got: %v", err)
	}

	// Test updating with incremented version (should succeed)
	// Create a new policy object to avoid modifying the stored one
	updatedPolicy := &storage.FaultInjectionPolicy{
		Metadata: storage.Metadata{
			Name:    "test-policy",
			Version: "1.0.1", // Incremented version
		},
		Spec: storage.PolicySpec{
			Rules: []storage.Rule{
				{
					Match: storage.MatchCondition{
						Method: &storage.StringMatcher{Exact: "GET"},
					},
					Fault: storage.FaultAction{
						Percentage: 100,
						Delay: &storage.DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}
	
	err = service.UpdatePolicy(updatedPolicy)
	if err != nil {
		t.Errorf("Expected successful update with incremented version, got: %v", err)
	}

	// Test updating again with same version (should fail)
	err = service.UpdatePolicy(updatedPolicy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for updating with same version again, got: %v", err)
	}

	// Test updating non-existent policy
	nonExistentPolicy := &storage.FaultInjectionPolicy{
		Metadata: storage.Metadata{
			Name:    "non-existent",
			Version: "1.0.0",
		},
		Spec: storage.PolicySpec{
			Rules: []storage.Rule{
				{
					Match: storage.MatchCondition{
						Path: &storage.PathMatcher{Exact: "/test"},
					},
					Fault: storage.FaultAction{
						Percentage: 50,
						Delay: &storage.DelayAction{
							FixedDelay: "1s",
						},
					},
				},
			},
		},
	}

	err = service.UpdatePolicy(nonExistentPolicy)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for updating non-existent policy, got: %v", err)
	}
}

func TestPolicyServiceInputValidation(t *testing.T) {
	store := storage.NewMemoryStore()
	service := NewPolicyService(store)

	// Test empty name in GetPolicyByName
	_, err := service.GetPolicyByName("")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for empty name, got: %v", err)
	}

	// Test whitespace-only name in GetPolicyByName
	_, err = service.GetPolicyByName("   ")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for whitespace-only name, got: %v", err)
	}

	// Test empty name in DeletePolicy
	err = service.DeletePolicy("")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for empty name in delete, got: %v", err)
	}

	// Test whitespace-only name in DeletePolicy
	err = service.DeletePolicy("   ")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Expected ErrInvalidInput for whitespace-only name in delete, got: %v", err)
	}
}

func TestDetailedError(t *testing.T) {
	err := NewDetailedError("validation_error", "Test message", map[string]interface{}{
		"field": "test_field",
		"value": "test_value",
	})

	if err.Type != "validation_error" {
		t.Errorf("Expected type 'validation_error', got: %s", err.Type)
	}

	if err.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got: %s", err.Message)
	}

	if err.Details["field"] != "test_field" {
		t.Errorf("Expected field 'test_field', got: %v", err.Details["field"])
	}

	expectedErrorString := "validation_error: Test message"
	if err.Error() != expectedErrorString {
		t.Errorf("Expected error string '%s', got: %s", expectedErrorString, err.Error())
	}
}
