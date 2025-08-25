package storage

import (
	"context"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestEtcdStore_BasicOperations tests basic CRUD operations
func TestEtcdStore_BasicOperations(t *testing.T) {
	// Skip if etcd is not available
	if !isEtcdAvailable() {
		t.Skip("etcd not available, skipping test")
	}

	store, err := NewEtcdStore([]string{"localhost:2379"})
	if err != nil {
		t.Fatalf("Failed to create etcd store: %v", err)
	}
	defer store.Close()

	// Test CreateOrUpdate
	policy := &FaultInjectionPolicy{
		Metadata: Metadata{Name: "test-policy"},
		Spec:     map[string]interface{}{"test": "value"},
	}

	err = store.CreateOrUpdate(policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Test Get
	retrievedPolicy, found := store.Get("test-policy")
	if !found {
		t.Fatalf("Policy not found after creation")
	}
	if retrievedPolicy.Metadata.Name != "test-policy" {
		t.Errorf("Expected name 'test-policy', got '%s'", retrievedPolicy.Metadata.Name)
	}

	// Test List
	policies := store.List()
	if len(policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(policies))
	}

	// Test Update
	policy.Spec = map[string]interface{}{"test": "updated_value"}
	err = store.CreateOrUpdate(policy)
	if err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// Verify update
	retrievedPolicy, found = store.Get("test-policy")
	if !found {
		t.Fatalf("Policy not found after update")
	}
	if spec, ok := retrievedPolicy.Spec.(map[string]interface{}); ok {
		if spec["test"] != "updated_value" {
			t.Errorf("Expected updated value 'updated_value', got '%v'", spec["test"])
		}
	} else {
		t.Errorf("Failed to cast spec to map[string]interface{}")
	}

	// Test Delete
	err = store.Delete("test-policy")
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify deletion
	_, found = store.Get("test-policy")
	if found {
		t.Errorf("Policy still found after deletion")
	}

	policies = store.List()
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies after deletion, got %d", len(policies))
	}
}

// TestEtcdStore_Watch tests the watch functionality
func TestEtcdStore_Watch(t *testing.T) {
	// Skip if etcd is not available
	if !isEtcdAvailable() {
		t.Skip("etcd not available, skipping test")
	}

	store, err := NewEtcdStore([]string{"localhost:2379"})
	if err != nil {
		t.Fatalf("Failed to create etcd store: %v", err)
	}
	defer store.Close()

	// Clean up any existing test policies
	store.Delete("watch-test-policy")

	// Start watching
	watchCh := store.Watch()

	// Create a policy
	policy := &FaultInjectionPolicy{
		Metadata: Metadata{Name: "watch-test-policy"},
		Spec:     map[string]interface{}{"test": "watch_value"},
	}

	go func() {
		time.Sleep(100 * time.Millisecond) // Give watch time to start
		store.CreateOrUpdate(policy)
	}()

	// Wait for the watch event
	select {
	case event := <-watchCh:
		if event.Type != EventTypePut {
			t.Errorf("Expected PUT event, got %s", event.Type)
		}
		if event.Policy.Metadata.Name != "watch-test-policy" {
			t.Errorf("Expected policy name 'watch-test-policy', got '%s'", event.Policy.Metadata.Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for watch event")
	}

	// Test delete event
	go func() {
		time.Sleep(100 * time.Millisecond)
		store.Delete("watch-test-policy")
	}()

	// Wait for the delete event
	select {
	case event := <-watchCh:
		if event.Type != EventTypeDelete {
			t.Errorf("Expected DELETE event, got %s", event.Type)
		}
		if event.Policy.Metadata.Name != "watch-test-policy" {
			t.Errorf("Expected policy name 'watch-test-policy', got '%s'", event.Policy.Metadata.Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for delete event")
	}
}

// isEtcdAvailable checks if etcd is available for testing
func isEtcdAvailable() bool {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Get(ctx, "test")
	return err == nil
}
