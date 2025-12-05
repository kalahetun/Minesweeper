package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"hfi/control-plane/logger"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	// policyPrefix is the etcd key prefix for all policies
	policyPrefix = "hfi/policies/"
	// defaultTimeout is the default timeout for etcd operations
	defaultTimeout = 5 * time.Second
)

// EtcdStore implements IPolicyStore using etcd as the backend storage.
type EtcdStore struct {
	client    *clientv3.Client
	ctx       context.Context
	cancel    context.CancelFunc
	watchers  []chan WatchEvent
	watcherMu sync.RWMutex
}

// NewEtcdStore creates a new EtcdStore instance.
func NewEtcdStore(endpoints []string) (*EtcdStore, error) {
	log := logger.WithComponent("storage.etcd")

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Error("Failed to create etcd client",
			zap.Strings("endpoints", endpoints),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	store := &EtcdStore{
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
		watchers: make([]chan WatchEvent, 0),
	}

	// Start the global watcher goroutine
	go store.startGlobalWatcher()

	log.Info("etcd store initialized successfully",
		zap.Strings("endpoints", endpoints))
	return store, nil
}

// Close closes the etcd connection and cancels all watchers.
func (e *EtcdStore) Close() error {
	e.cancel()
	e.watcherMu.Lock()
	for _, watcher := range e.watchers {
		close(watcher)
	}
	e.watchers = nil
	e.watcherMu.Unlock()
	return e.client.Close()
}

// HealthCheck verifies the etcd connection is healthy by performing a simple get operation.
func (e *EtcdStore) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Perform a simple status check on the etcd cluster
	_, err := e.client.Status(ctx, e.client.Endpoints()[0])
	if err != nil {
		return fmt.Errorf("etcd health check failed: %w", err)
	}
	return nil
}

// policyKey returns the etcd key for a given policy name.
func (e *EtcdStore) policyKey(name string) string {
	return policyPrefix + name
}

// CreateOrUpdate creates a new policy or updates an existing one atomically.
func (e *EtcdStore) CreateOrUpdate(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return fmt.Errorf("policy or policy name cannot be empty")
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	key := e.policyKey(policy.Metadata.Name)

	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	// Use a transaction to ensure atomicity
	txn := e.client.Txn(ctx)
	_, err = txn.Then(clientv3.OpPut(key, string(data))).Commit()
	if err != nil {
		return fmt.Errorf("failed to put policy to etcd: %w", err)
	}

	return nil
}

// Create creates a new policy. Returns ErrAlreadyExists if the policy already exists.
func (e *EtcdStore) Create(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return ErrInvalidInput
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	key := e.policyKey(policy.Metadata.Name)

	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	// Use etcd transaction to check if key doesn't exist before creating
	txn := e.client.Txn(ctx)
	resp, err := txn.If(
		// Condition: key doesn't exist (CreateRevision == 0)
		clientv3.Compare(clientv3.CreateRevision(key), "=", 0),
	).Then(
		// If condition is true: create the key
		clientv3.OpPut(key, string(data)),
	).Else(
		// If condition is false: key already exists
		clientv3.OpGet(key),
	).Commit()

	if err != nil {
		return fmt.Errorf("failed to create policy in etcd: %w", err)
	}

	// If the transaction condition was false, the key already exists
	if !resp.Succeeded {
		return ErrAlreadyExists
	}

	return nil
}

// Update updates an existing policy. Returns ErrNotFound if the policy doesn't exist.
func (e *EtcdStore) Update(policy *FaultInjectionPolicy) error {
	if policy == nil || policy.Metadata.Name == "" {
		return ErrInvalidInput
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	key := e.policyKey(policy.Metadata.Name)

	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	// Use etcd transaction to check if key exists before updating
	txn := e.client.Txn(ctx)
	resp, err := txn.If(
		// Condition: key exists (CreateRevision > 0)
		clientv3.Compare(clientv3.CreateRevision(key), ">", 0),
	).Then(
		// If condition is true: update the key
		clientv3.OpPut(key, string(data)),
	).Else(
		// If condition is false: key doesn't exist
		clientv3.OpGet(key),
	).Commit()

	if err != nil {
		return fmt.Errorf("failed to update policy in etcd: %w", err)
	}

	// If the transaction condition was false, the key doesn't exist
	if !resp.Succeeded {
		return ErrNotFound
	}

	return nil
}

// Get retrieves a policy by its name.
func (e *EtcdStore) Get(name string) (*FaultInjectionPolicy, error) {
	if name == "" {
		return nil, ErrNotFound
	}

	key := e.policyKey(name)

	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	var policy FaultInjectionPolicy
	if err := json.Unmarshal(resp.Kvs[0].Value, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	return &policy, nil
}

// Delete removes a policy by its name.
// Uses a transaction to ensure atomicity.
func (e *EtcdStore) Delete(name string) error {
	if name == "" {
		return ErrInvalidInput
	}

	key := e.policyKey(name)

	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	// Use etcd transaction to check if key exists before deleting
	txn := e.client.Txn(ctx)
	resp, err := txn.If(
		// Condition: key exists (CreateRevision > 0)
		clientv3.Compare(clientv3.CreateRevision(key), ">", 0),
	).Then(
		// If condition is true: delete the key
		clientv3.OpDelete(key),
	).Else(
		// If condition is false: key doesn't exist
		clientv3.OpGet(key),
	).Commit()

	if err != nil {
		return fmt.Errorf("failed to delete policy from etcd: %w", err)
	}

	// If the transaction condition was false, the key doesn't exist
	if !resp.Succeeded {
		return ErrNotFound
	}

	return nil
} // List retrieves all policies.
func (e *EtcdStore) List() []*FaultInjectionPolicy {
	// Create a context with timeout for this operation
	ctx, cancel := context.WithTimeout(e.ctx, defaultTimeout)
	defer cancel()

	resp, err := e.client.Get(ctx, policyPrefix, clientv3.WithPrefix())
	if err != nil {
		return []*FaultInjectionPolicy{}
	}

	policies := make([]*FaultInjectionPolicy, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var policy FaultInjectionPolicy
		if err := json.Unmarshal(kv.Value, &policy); err != nil {
			// Log error but continue with other policies
			continue
		}
		policies = append(policies, &policy)
	}

	return policies
}

// Watch returns a channel that receives notifications of policy changes.
func (e *EtcdStore) Watch() <-chan WatchEvent {
	ch := make(chan WatchEvent, 100) // Buffer to prevent blocking

	e.watcherMu.Lock()
	e.watchers = append(e.watchers, ch)
	e.watcherMu.Unlock()

	return ch
}

// WatchWithContext returns a channel that receives notifications until ctx is canceled.
// The caller can stop watching by canceling the context.
func (e *EtcdStore) WatchWithContext(ctx context.Context) <-chan WatchEvent {
	ch := make(chan WatchEvent, 100)

	e.watcherMu.Lock()
	e.watchers = append(e.watchers, ch)
	e.watcherMu.Unlock()

	// Launch a goroutine that removes and closes the channel when context is canceled
	go func() {
		<-ctx.Done()
		// Remove and close the watcher when context is canceled
		e.watcherMu.Lock()
		defer e.watcherMu.Unlock()

		// Find and remove this watcher from the list
		for i, watcher := range e.watchers {
			if watcher == ch {
				// Remove by swapping with last element and truncating
				e.watchers[i] = e.watchers[len(e.watchers)-1]
				e.watchers = e.watchers[:len(e.watchers)-1]
				close(ch)
				break
			}
		}
	}()

	return ch
}

// startGlobalWatcher starts a global watcher that monitors all policy changes
// and distributes events to all registered watchers.
func (e *EtcdStore) startGlobalWatcher() {
	watchCh := e.client.Watch(e.ctx, policyPrefix, clientv3.WithPrefix())

	for {
		select {
		case <-e.ctx.Done():
			return
		case wresp, ok := <-watchCh:
			if !ok {
				// Watch channel closed, recreate it
				watchCh = e.client.Watch(e.ctx, policyPrefix, clientv3.WithPrefix())
				continue
			}

			if wresp.Err() != nil {
				// Log error and continue watching
				continue
			}

			// Process all events in this watch response
			for _, event := range wresp.Events {
				e.processWatchEvent(event)
			}
		}
	}
}

// processWatchEvent converts an etcd event to our WatchEvent and distributes it.
func (e *EtcdStore) processWatchEvent(event *clientv3.Event) {
	var watchEvent WatchEvent
	var policy *FaultInjectionPolicy

	// Extract policy name from key
	key := string(event.Kv.Key)
	if !strings.HasPrefix(key, policyPrefix) {
		return
	}

	switch event.Type {
	case mvccpb.PUT:
		// Parse the policy from the event value
		var p FaultInjectionPolicy
		if err := json.Unmarshal(event.Kv.Value, &p); err != nil {
			return // Skip malformed policies
		}
		policy = &p
		watchEvent = WatchEvent{
			Type:   EventTypePut,
			Policy: policy,
		}
	case mvccpb.DELETE:
		// For delete events, we only have the key, not the full policy
		// Extract the policy name from the key
		policyName := strings.TrimPrefix(key, policyPrefix)
		policy = &FaultInjectionPolicy{
			Metadata: Metadata{Name: policyName},
		}
		watchEvent = WatchEvent{
			Type:   EventTypeDelete,
			Policy: policy,
		}
	default:
		return // Ignore other event types
	}

	// Distribute the event to all watchers
	e.distributeWatchEvent(watchEvent)
}

// distributeWatchEvent sends the watch event to all registered watchers.
// Uses a copy-on-read pattern to avoid holding the lock during sends.
func (e *EtcdStore) distributeWatchEvent(event WatchEvent) {
	// Make a copy of watchers to avoid holding lock during sends
	e.watcherMu.RLock()
	watchers := make([]chan WatchEvent, len(e.watchers))
	copy(watchers, e.watchers)
	e.watcherMu.RUnlock()

	// Track dead watchers to clean up later
	var deadIndices []int

	// Send to all watchers without holding the lock
	for i, watcher := range watchers {
		select {
		case watcher <- event:
			// Event sent successfully
		default:
			// Watcher channel is full or closed, mark for cleanup
			deadIndices = append(deadIndices, i)
		}
	}

	// Cleanup dead watchers
	if len(deadIndices) > 0 {
		e.cleanupDeadWatchers(deadIndices)
	}
}

// cleanupDeadWatchers removes watchers that failed to receive events.
// This is called after we've released the read lock.
func (e *EtcdStore) cleanupDeadWatchers(deadIndices []int) {
	e.watcherMu.Lock()
	defer e.watcherMu.Unlock()

	// Build a new slice without the dead watchers
	// We need to be careful because indices may have changed due to concurrent adds
	var newWatchers []chan WatchEvent

	deadSet := make(map[int]bool)
	for _, idx := range deadIndices {
		deadSet[idx] = true
	}

	for i, watcher := range e.watchers {
		if deadSet[i] {
			// Try to close the dead watcher, ignore if already closed
			select {
			case <-watcher:
				// Channel already closed or can receive
			default:
				// Try to close
				close(watcher)
			}
		} else {
			// Keep this watcher
			newWatchers = append(newWatchers, watcher)
		}
	}

	e.watchers = newWatchers
}
