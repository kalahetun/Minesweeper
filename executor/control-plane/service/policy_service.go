package service

import (
	"errors"
	"hfi/control-plane/storage"
	"strings"
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

// CreateOrUpdatePolicy creates or updates a policy with validation.
func (s *PolicyService) CreateOrUpdatePolicy(policy *storage.FaultInjectionPolicy) error {
	// Validate the policy first
	if err := s.ValidatePolicy(policy); err != nil {
		return errors.Join(ErrInvalidInput, err)
	}

	return s.store.CreateOrUpdate(policy)
}

// CreatePolicy creates a new policy with validation.
func (s *PolicyService) CreatePolicy(policy *storage.FaultInjectionPolicy) error {
	// Enhanced validation for create operation
	if err := s.validatePolicyForCreate(policy); err != nil {
		return errors.Join(ErrInvalidInput, err)
	}

	return s.store.Create(policy)
}

// UpdatePolicy updates an existing policy with validation.
func (s *PolicyService) UpdatePolicy(policy *storage.FaultInjectionPolicy) error {
	// Enhanced validation for update operation
	if err := s.validatePolicyForUpdate(policy); err != nil {
		return errors.Join(ErrInvalidInput, err)
	}

	return s.store.Update(policy)
}

// GetPolicyByName retrieves a policy by its name.
func (s *PolicyService) GetPolicyByName(name string) (*storage.FaultInjectionPolicy, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidInput
	}
	return s.store.Get(name)
}

// ListPolicies retrieves all policies.
func (s *PolicyService) ListPolicies() []*storage.FaultInjectionPolicy {
	return s.store.List()
}

// DeletePolicy deletes a policy by its name.
func (s *PolicyService) DeletePolicy(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidInput
	}
	return s.store.Delete(name)
}

// validatePolicyForCreate performs additional validation for policy creation.
func (s *PolicyService) validatePolicyForCreate(policy *storage.FaultInjectionPolicy) error {
	// Basic validation first
	if err := s.ValidatePolicy(policy); err != nil {
		return err
	}

	// Additional create-specific validations
	if err := s.validatePolicyName(policy.Metadata.Name); err != nil {
		return err
	}

	if err := s.validatePolicyVersion(policy.Metadata.Version); err != nil {
		return err
	}

	return nil
}

// validatePolicyForUpdate performs additional validation for policy updates.
func (s *PolicyService) validatePolicyForUpdate(policy *storage.FaultInjectionPolicy) error {
	// Basic validation first
	if err := s.ValidatePolicy(policy); err != nil {
		return err
	}

	// Check if the policy exists in the store for update
	existingPolicy, err := s.store.Get(policy.Metadata.Name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return NewDetailedError("validation_error", "cannot update non-existent policy", 
				map[string]interface{}{
					"policy_name": policy.Metadata.Name,
					"operation": "update",
				})
		}
		return err
	}

	// Version validation for updates
	if err := s.validateVersionUpdate(existingPolicy.Metadata.Version, policy.Metadata.Version); err != nil {
		return err
	}

	return nil
}

// validatePolicyName validates the policy name format and constraints.
func (s *PolicyService) validatePolicyName(name string) error {
	if len(name) < 3 {
		return NewDetailedError("validation_error", "policy name too short", 
			map[string]interface{}{
				"field": "metadata.name",
				"min_length": 3,
				"actual_length": len(name),
			})
	}

	if len(name) > 63 {
		return NewDetailedError("validation_error", "policy name too long", 
			map[string]interface{}{
				"field": "metadata.name",
				"max_length": 63,
				"actual_length": len(name),
			})
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return NewDetailedError("validation_error", "policy name contains invalid characters", 
				map[string]interface{}{
					"field": "metadata.name",
					"invalid_char": string(r),
					"position": i,
					"allowed_chars": "a-z, A-Z, 0-9, -, _",
				})
		}
	}

	// Name cannot start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return NewDetailedError("validation_error", "policy name cannot start or end with hyphen", 
			map[string]interface{}{
				"field": "metadata.name",
				"value": name,
			})
	}

	return nil
}

// validatePolicyVersion validates the policy version format.
func (s *PolicyService) validatePolicyVersion(version string) error {
	if version == "" {
		return NewDetailedError("validation_error", "policy version is required", 
			map[string]interface{}{
				"field": "metadata.version",
			})
	}

	// Simple semantic version validation (e.g., "1.0.0")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return NewDetailedError("validation_error", "policy version must follow semantic versioning (x.y.z)", 
			map[string]interface{}{
				"field": "metadata.version",
				"value": version,
				"expected_format": "x.y.z",
			})
	}

	for i, part := range parts {
		if part == "" {
			return NewDetailedError("validation_error", "policy version parts cannot be empty", 
				map[string]interface{}{
					"field": "metadata.version",
					"value": version,
					"empty_part_index": i,
				})
		}

		// Check if part is numeric
		for _, r := range part {
			if r < '0' || r > '9' {
				return NewDetailedError("validation_error", "policy version parts must be numeric", 
					map[string]interface{}{
						"field": "metadata.version",
						"value": version,
						"invalid_part": part,
						"part_index": i,
					})
			}
		}
	}

	return nil
}

// validateVersionUpdate validates version changes during updates.
func (s *PolicyService) validateVersionUpdate(oldVersion, newVersion string) error {
	if oldVersion == newVersion {
		return NewDetailedError("validation_error", "version must be incremented for updates", 
			map[string]interface{}{
				"field": "metadata.version",
				"old_version": oldVersion,
				"new_version": newVersion,
			})
	}

	// In a real implementation, you might want to validate that the new version
	// is actually greater than the old version using semantic version comparison
	
	return nil
}
