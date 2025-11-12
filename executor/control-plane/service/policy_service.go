package service

import (
	"errors"
	"hfi/control-plane/storage"
	"strings"
	"time"
)

// PolicyService provides business logic for policy operations.
type PolicyService struct {
	store              storage.IPolicyStore
	expirationRegistry *ExpirationRegistry
}

// NewPolicyService creates a new PolicyService instance.
func NewPolicyService(store storage.IPolicyStore) *PolicyService {
	return &PolicyService{
		store:              store,
		expirationRegistry: NewExpirationRegistry(),
	}
}

// CreateOrUpdatePolicy creates or updates a policy with validation.
func (s *PolicyService) CreateOrUpdatePolicy(policy *storage.FaultInjectionPolicy) error {
	// Validate the policy first
	if err := s.ValidatePolicy(policy); err != nil {
		return errors.Join(ErrInvalidInput, err)
	}

	// Log time control configuration if specified
	s.logTimeControlConfig(policy)

	// Store the policy
	if err := s.store.CreateOrUpdate(policy); err != nil {
		return err
	}

	// Register auto-expiration if duration_seconds is specified and > 0
	s.scheduleAutoExpiration(policy)

	return nil
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

	// Cancel any scheduled auto-deletion timer for this policy
	s.expirationRegistry.Cancel(name)

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
					"operation":   "update",
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
				"field":         "metadata.name",
				"min_length":    3,
				"actual_length": len(name),
			})
	}

	if len(name) > 63 {
		return NewDetailedError("validation_error", "policy name too long",
			map[string]interface{}{
				"field":         "metadata.name",
				"max_length":    63,
				"actual_length": len(name),
			})
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return NewDetailedError("validation_error", "policy name contains invalid characters",
				map[string]interface{}{
					"field":         "metadata.name",
					"invalid_char":  string(r),
					"position":      i,
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
				"field":           "metadata.version",
				"value":           version,
				"expected_format": "x.y.z",
			})
	}

	for i, part := range parts {
		if part == "" {
			return NewDetailedError("validation_error", "policy version parts cannot be empty",
				map[string]interface{}{
					"field":            "metadata.version",
					"value":            version,
					"empty_part_index": i,
				})
		}

		// Check if part is numeric
		for _, r := range part {
			if r < '0' || r > '9' {
				return NewDetailedError("validation_error", "policy version parts must be numeric",
					map[string]interface{}{
						"field":        "metadata.version",
						"value":        version,
						"invalid_part": part,
						"part_index":   i,
					})
			}
		}
	}

	return nil
}

// validateVersionUpdate validates version changes during updates.
// Ensures new version is greater than old version using semantic versioning.
func (s *PolicyService) validateVersionUpdate(oldVersion, newVersion string) error {
	if oldVersion == newVersion {
		return NewDetailedError("validation_error", "version must be incremented for updates",
			map[string]interface{}{
				"field":       "metadata.version",
				"old_version": oldVersion,
				"new_version": newVersion,
			})
	}

	// Parse both versions to ensure they're valid semantic versions
	oldParts := strings.Split(oldVersion, ".")
	newParts := strings.Split(newVersion, ".")

	if len(oldParts) != 3 || len(newParts) != 3 {
		return NewDetailedError("validation_error",
			"both old and new versions must be semantic versions (x.y.z)",
			map[string]interface{}{
				"field":       "metadata.version",
				"old_version": oldVersion,
				"new_version": newVersion,
			})
	}

	// Parse version numbers for comparison
	oldMajor, _ := parseInt(oldParts[0])
	oldMinor, _ := parseInt(oldParts[1])
	oldPatch, _ := parseInt(oldParts[2])

	newMajor, _ := parseInt(newParts[0])
	newMinor, _ := parseInt(newParts[1])
	newPatch, _ := parseInt(newParts[2])

	// Compare versions: major.minor.patch
	// New version must be strictly greater than old version
	if newMajor > oldMajor {
		return nil // Major version increased, valid
	}
	if newMajor == oldMajor && newMinor > oldMinor {
		return nil // Major same, minor increased, valid
	}
	if newMajor == oldMajor && newMinor == oldMinor && newPatch > oldPatch {
		return nil // Major and minor same, patch increased, valid
	}

	// Version not increased or decreased
	return NewDetailedError("validation_error",
		"new version must be greater than old version",
		map[string]interface{}{
			"field":       "metadata.version",
			"old_version": oldVersion,
			"new_version": newVersion,
			"comparison":  "new must be > old",
		})
}

// parseInt parses a version part to integer, returns 0 if invalid
func parseInt(s string) (int, error) {
	result := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("non-numeric")
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
}

// logTimeControlConfig logs the time control configuration of a policy.
// This helps track when policies have automatic expiration enabled.
func (s *PolicyService) logTimeControlConfig(policy *storage.FaultInjectionPolicy) {
	if policy == nil || len(policy.Spec.Rules) == 0 {
		return
	}

	for i, rule := range policy.Spec.Rules {
		fault := rule.Fault

		// Only log if time control fields are explicitly set
		if fault.StartDelayMs > 0 || fault.DurationSeconds > 0 {
			logMsg := "Policy time control configured"

			// Format lifecycle type
			lifecycleType := "persistent"
			if fault.DurationSeconds > 0 {
				lifecycleType = "temporary"
			}

			// Log with details
			// Logger would go here when proper logger is added to service
			_ = logMsg // Placeholder for future logger integration
			_ = lifecycleType
			_ = i

			// Example log content (for documentation):
			// fmt.Printf("Policy: %s, Rule[%d], Type: %s, StartDelayMs: %d, DurationSeconds: %d\n",
			//     policy.Metadata.Name, i, lifecycleType, fault.StartDelayMs, fault.DurationSeconds)
		}
	}
}

// scheduleAutoExpiration schedules a policy for automatic deletion if it has a duration_seconds > 0
// This is called after a policy is successfully stored
func (s *PolicyService) scheduleAutoExpiration(policy *storage.FaultInjectionPolicy) {
	if policy == nil || len(policy.Spec.Rules) == 0 {
		return
	}

	// Check the first rule's fault action for duration_seconds
	// In most cases, all rules in a policy have the same lifecycle
	durationSeconds := policy.Spec.Rules[0].Fault.DurationSeconds

	// Only schedule if duration_seconds is explicitly set and > 0
	if durationSeconds <= 0 {
		// Cancel any existing timer for this policy (in case it was previously temporary)
		s.expirationRegistry.Cancel(policy.Metadata.Name)
		return
	}

	// Register the auto-deletion
	duration := time.Duration(durationSeconds) * time.Second
	s.expirationRegistry.Register(
		policy.Metadata.Name,
		duration,
		func() error {
			// Delete the policy when the timer expires
			return s.DeletePolicy(policy.Metadata.Name)
		},
	)
}
