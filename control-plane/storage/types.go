package storage

// Metadata contains common metadata for all policy types.
type Metadata struct {
	Name string `json:"name"`
}

// FaultInjectionPolicy defines the structure for a fault injection policy.
type FaultInjectionPolicy struct {
	Metadata Metadata   `json:"metadata"`
	Spec     PolicySpec `json:"spec"`
}

// PolicySpec defines the specification for a fault injection policy.
type PolicySpec struct {
	Rules []Rule `json:"rules"`
}

// Rule defines a single fault injection rule.
type Rule struct {
	Match MatchCondition `json:"match"`
	Fault FaultAction    `json:"fault"`
}

// MatchCondition defines the conditions for when to apply fault injection.
type MatchCondition struct {
	Method  *StringMatcher  `json:"method,omitempty"`
	Path    *PathMatcher    `json:"path,omitempty"`
	Headers []HeaderMatcher `json:"headers,omitempty"`
}

// StringMatcher defines string matching options.
type StringMatcher struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// PathMatcher defines path matching options.
type PathMatcher struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// HeaderMatcher defines header matching options.
type HeaderMatcher struct {
	Name   string `json:"name"`
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// FaultAction defines the fault injection actions to apply.
type FaultAction struct {
	Percentage int          `json:"percentage"`
	Delay      *DelayAction `json:"delay,omitempty"`
	Abort      *AbortAction `json:"abort,omitempty"`
}

// DelayAction defines a delay fault injection action.
type DelayAction struct {
	FixedDelay string `json:"fixed_delay"`
}

// AbortAction defines an abort fault injection action.
type AbortAction struct {
	HTTPStatus int `json:"httpStatus"`
}
