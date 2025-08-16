package storage

// Metadata contains common metadata for all policy types.
type Metadata struct {
	Name string `json:"name"`
}

// FaultInjectionPolicy defines the structure for a fault injection policy.
// For the MVP, Spec is kept as an interface{} for simplicity.
type FaultInjectionPolicy struct {
	Metadata Metadata    `json:"metadata"`
	Spec     interface{} `json:"spec"`
}
