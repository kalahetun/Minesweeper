package storage

// EventType defines the type of a watch event.
type EventType string

const (
	// EventTypePut represents a create or update event.
	EventTypePut EventType = "PUT"
	// EventTypeDelete represents a delete event.
	EventTypeDelete EventType = "DELETE"
)

// WatchEvent represents a change in the policy store.
type WatchEvent struct {
	Type   EventType
	Policy *FaultInjectionPolicy
}

// IPolicyStore defines the interface for policy storage.
type IPolicyStore interface {
	// CreateOrUpdate creates a new policy or updates an existing one.
	CreateOrUpdate(policy *FaultInjectionPolicy) error
	// Get retrieves a policy by its name.
	Get(name string) (*FaultInjectionPolicy, error)
	// Delete removes a policy by its name.
	Delete(name string) error
	// List retrieves all policies.
	List() []*FaultInjectionPolicy
	// Watch returns a channel that receives notifications of policy changes.
	Watch() <-chan WatchEvent
}
