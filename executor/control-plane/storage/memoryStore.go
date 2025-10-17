package storage

import (
	"fmt"
	"sync"
)

// memoryStore is an in-memory implementation of IPolicyStore.
type memoryStore struct {
	mu          sync.RWMutex
	policies    map[string]*FaultInjectionPolicy
	watchers    map[int]chan WatchEvent
	nextWatcher int
}

// NewMemoryStore creates a new in-memory policy store.
func NewMemoryStore() IPolicyStore {
	return &memoryStore{
		policies: make(map[string]*FaultInjectionPolicy),
		watchers: make(map[int]chan WatchEvent),
	}
}

// CreateOrUpdate creates a new policy or updates an existing one.
func (s *memoryStore) CreateOrUpdate(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return fmt.Errorf("policy and policy name must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.policies[policy.Metadata.Name] = policy
	s.broadcast(WatchEvent{Type: EventTypePut, Policy: policy})
	return nil
}

// Create creates a new policy. Returns ErrAlreadyExists if the policy already exists.
func (s *memoryStore) Create(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[policy.Metadata.Name]; exists {
		return ErrAlreadyExists
	}

	s.policies[policy.Metadata.Name] = policy
	s.broadcast(WatchEvent{Type: EventTypePut, Policy: policy})
	return nil
}

// Update updates an existing policy. Returns ErrNotFound if the policy doesn't exist.
func (s *memoryStore) Update(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[policy.Metadata.Name]; !exists {
		return ErrNotFound
	}

	s.policies[policy.Metadata.Name] = policy
	s.broadcast(WatchEvent{Type: EventTypePut, Policy: policy})
	return nil
}

// Get retrieves a policy by its name.
func (s *memoryStore) Get(name string) (*FaultInjectionPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.policies[name]
	if !ok {
		return nil, ErrNotFound
	}
	return policy, nil
}

// Delete removes a policy by its name.
func (s *memoryStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policy, ok := s.policies[name]; ok {
		delete(s.policies, name)
		s.broadcast(WatchEvent{Type: EventTypeDelete, Policy: policy})
		return nil
	}
	return ErrNotFound
}

// List retrieves all policies.
func (s *memoryStore) List() []*FaultInjectionPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*FaultInjectionPolicy, 0, len(s.policies))
	for _, policy := range s.policies {
		list = append(list, policy)
	}
	return list
}

// Watch returns a channel that receives notifications of policy changes.
func (s *memoryStore) Watch() <-chan WatchEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new watcher channel
	watcherChan := make(chan WatchEvent, 100) // Buffered channel to avoid blocking
	watcherID := s.nextWatcher
	s.nextWatcher++

	s.watchers[watcherID] = watcherChan

	// Return a read-only channel to the caller
	return watcherChan
}

// broadcast sends a watch event to all registered watchers.
// This should be called within a lock.
func (s *memoryStore) broadcast(event WatchEvent) {
	for id, ch := range s.watchers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel is blocked, assume the watcher is dead or slow.
			// Close the channel and remove it.
			close(ch)
			delete(s.watchers, id)
		}
	}
}
