package types

// FaultInjectionPolicy represents a fault injection policy
// Matches the Control Plane's simple structure
type FaultInjectionPolicy struct {
	Metadata PolicyMetadata `json:"metadata" yaml:"metadata"`
	Spec     interface{}    `json:"spec" yaml:"spec"`
}

// PolicyMetadata contains metadata for the policy
type PolicyMetadata struct {
	Name string `json:"name" yaml:"name"`
}
