package types

// FaultInjectionPolicy represents a fault injection policy
type FaultInjectionPolicy struct {
	Metadata PolicyMetadata `json:"metadata" yaml:"metadata"`
	Spec     PolicySpec     `json:"spec" yaml:"spec"`
}

// PolicyMetadata contains metadata for the policy
type PolicyMetadata struct {
	Name string `json:"name" yaml:"name"`
}

// PolicySpec contains the specification for fault injection
type PolicySpec struct {
	Rules []Rule `json:"rules" yaml:"rules"`
}

// Rule represents a single fault injection rule
type Rule struct {
	Match MatchCondition `json:"match" yaml:"match"`
	Fault FaultAction    `json:"fault" yaml:"fault"`
}

// MatchCondition defines when a fault should be applied
type MatchCondition struct {
	Method  *StringMatcher   `json:"method,omitempty" yaml:"method,omitempty"`
	Path    *PathMatcher     `json:"path,omitempty" yaml:"path,omitempty"`
	Headers []HeaderMatcher  `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// StringMatcher matches strings
type StringMatcher struct {
	Exact  string `json:"exact,omitempty" yaml:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// PathMatcher matches request paths
type PathMatcher struct {
	Exact  string `json:"exact,omitempty" yaml:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// HeaderMatcher matches request headers
type HeaderMatcher struct {
	Name   string `json:"name" yaml:"name"`
	Exact  string `json:"exact,omitempty" yaml:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// FaultAction defines what fault to inject
type FaultAction struct {
	Percentage int          `json:"percentage" yaml:"percentage"`
	Delay      *DelayAction `json:"delay,omitempty" yaml:"delay,omitempty"`
	Abort      *AbortAction `json:"abort,omitempty" yaml:"abort,omitempty"`
}

// DelayAction injects delay
type DelayAction struct {
	FixedDelay string `json:"fixed_delay" yaml:"fixed_delay"`
}

// AbortAction aborts requests
type AbortAction struct {
	HTTPStatus int `json:"httpStatus" yaml:"httpStatus"`
}
