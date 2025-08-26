package service

import (
	"hfi/control-plane/storage"
)

// PolicyService provides business logic for policy operations.
type PolicyService struct {
	store storage.IPolicyStore
}

// NewPolicyService creates a new PolicyService instance.
func NewPolicyService(store storage.IPolicyStore) *PolicyService {
	return &PolicyService{
		store: store,
	}
}

// CreateOrUpdatePolicy creates or updates a policy.
func (s *PolicyService) CreateOrUpdatePolicy(policy *storage.FaultInjectionPolicy) error {
	return s.store.CreateOrUpdate(policy)
}

// GetPolicyByName retrieves a policy by its name.
func (s *PolicyService) GetPolicyByName(name string) (*storage.FaultInjectionPolicy, error) {
	return s.store.Get(name)
}

// ListPolicies retrieves all policies.
func (s *PolicyService) ListPolicies() []*storage.FaultInjectionPolicy {
	return s.store.List()
}

// DeletePolicy deletes a policy by its name.
func (s *PolicyService) DeletePolicy(name string) error {
	return s.store.Delete(name)
}
