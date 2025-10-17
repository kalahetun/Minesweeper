package service

import (
	"fmt"
	"hfi/control-plane/storage"
)

// ValidatePolicy validates a fault injection policy.
func (s *PolicyService) ValidatePolicy(policy *storage.FaultInjectionPolicy) error {
	if policy == nil {
		return &ValidationError{Field: "policy", Message: "policy cannot be nil"}
	}

	if policy.Metadata.Name == "" {
		return &ValidationError{Field: "metadata.name", Message: "policy name is required"}
	}

	if len(policy.Spec.Rules) == 0 {
		return &ValidationError{Field: "spec.rules", Message: "at least one rule is required"}
	}

	// Validate each rule
	for i, rule := range policy.Spec.Rules {
		if err := s.validateRule(rule, fmt.Sprintf("spec.rules[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// validateRule validates a single rule within a policy.
func (s *PolicyService) validateRule(rule storage.Rule, fieldPath string) error {
	// Validate match conditions
	hasMatchCondition := false
	if rule.Match.Method != nil && (rule.Match.Method.Exact != "" || rule.Match.Method.Prefix != "" || rule.Match.Method.Regex != "") {
		hasMatchCondition = true
	}
	if rule.Match.Path != nil && (rule.Match.Path.Exact != "" || rule.Match.Path.Prefix != "" || rule.Match.Path.Regex != "") {
		hasMatchCondition = true
	}
	if len(rule.Match.Headers) > 0 {
		hasMatchCondition = true
	}

	if !hasMatchCondition {
		return &ValidationError{Field: fieldPath + ".match", Message: "at least one match condition is required"}
	}

	// Validate fault configuration
	hasFaultAction := false
	if rule.Fault.Delay != nil && rule.Fault.Delay.FixedDelay != "" {
		hasFaultAction = true
	}
	if rule.Fault.Abort != nil && rule.Fault.Abort.HTTPStatus > 0 {
		hasFaultAction = true
	}

	if !hasFaultAction {
		return &ValidationError{Field: fieldPath + ".fault", Message: "at least one fault action is required"}
	}

	// Validate percentage
	if rule.Fault.Percentage < 0 || rule.Fault.Percentage > 100 {
		return &ValidationError{Field: fieldPath + ".fault.percentage", Message: "percentage must be between 0 and 100"}
	}

	return nil
}
