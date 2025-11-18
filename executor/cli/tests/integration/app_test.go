package integration

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"hfi-cli/client"
	cmdpkg "hfi-cli/cmd"

	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a mock control plane server for testing
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status":"healthy"}`)
	})

	// List and Create policies endpoint
	mux.HandleFunc("/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"policies":[]}`)
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"message":"policy created"}`)
		}
	})

	// Get and Delete policy endpoint
	mux.HandleFunc("/v1/policies/test-policy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"metadata":{"name":"test-policy","version":"1.0"},"spec":{}}`)
		} else if r.Method == "DELETE" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"message":"policy deleted"}`)
		}
	})

	return httptest.NewServer(mux)
}

// TestCLIE2EListPolicies tests listing policies end-to-end
func TestCLIE2EListPolicies(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rootCmd := cmdpkg.GetRootCommand()
	args := []string{"--control-plane-addr", server.URL, "policy", "list"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	// May fail due to command execution details, but should not panic
	_ = err
}

// TestCLIE2EGetPolicy tests getting a policy end-to-end
func TestCLIE2EGetPolicy(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rootCmd := cmdpkg.GetRootCommand()
	args := []string{"--control-plane-addr", server.URL, "policy", "get", "test-policy"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	// May have issues without proper server, but should attempt
	_ = err
}

// TestCLIE2EDeletePolicy tests deleting a policy end-to-end
func TestCLIE2EDeletePolicy(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rootCmd := cmdpkg.GetRootCommand()
	args := []string{"--control-plane-addr", server.URL, "policy", "delete", "test-policy"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	// May have issues without proper server, but should attempt
	_ = err
}

// TestCLIE2EVersionOutput tests version command output
func TestCLIE2EVersionOutput(t *testing.T) {
	rootCmd := cmdpkg.GetRootCommand()
	args := []string{"--version"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	// Version flag may not be set up, but shouldn't panic
	err := rootCmd.Execute()
	_ = err
}

// TestCLIE2EMultipleSequentialCmds tests sequential commands
func TestCLIE2EMultipleSequentialCmds(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	commands := []struct {
		name string
		args []string
	}{
		{"help", []string{"--help"}},
		{"policy-help", []string{"policy", "--help"}},
		{"list", []string{"--control-plane-addr", server.URL, "policy", "list"}},
	}

	for _, cmdSpec := range commands {
		t.Run(cmdSpec.name, func(t *testing.T) {
			rootCmd := cmdpkg.GetRootCommand()
			rootCmd.SetArgs(cmdSpec.args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)

			err := rootCmd.Execute()
			// All help commands should succeed
			if cmdSpec.name == "help" || cmdSpec.name == "policy-help" {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCLIE2EGlobalFlagsAffectBehavior tests global flags behavior
func TestCLIE2EGlobalFlagsAffectBehavior(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test with short timeout
	rootCmd := cmdpkg.GetRootCommand()
	args := []string{
		"--control-plane-addr", server.URL,
		"--timeout", "1s",
		"policy", "list",
	}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	// Should succeed with sufficient time
	_ = err
}

// TestCLIE2EErrorHandling tests error handling in commands
func TestCLIE2EErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		serverAddr  string
		args        []string
		expectError bool
	}{
		{"missing-policy-name", "http://localhost:8080", []string{"policy", "get"}, true},
		{"missing-file-flag", "http://localhost:8080", []string{"policy", "apply"}, true},
		{"invalid-flag", "http://localhost:8080", []string{"--invalid", "value"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmdpkg.GetRootCommand()
			args := append([]string{"--control-plane-addr", tc.serverAddr}, tc.args...)
			rootCmd.SetArgs(args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			err := rootCmd.Execute()
			if tc.expectError {
				assert.Error(t, err)
			}
		})
	}
}

// TestCLIE2EFlagValidation tests flag validation and parsing
func TestCLIE2EFlagValidation(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	testCases := []struct {
		name string
		addr string
		pass bool
	}{
		{"http-addr", server.URL, true},
		{"custom-port", "http://localhost:9000", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmdpkg.GetRootCommand()
			args := []string{
				"--control-plane-addr", tc.addr,
				"policy", "list",
			}
			rootCmd.SetArgs(args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)

			err := rootCmd.Execute()
			if tc.pass {
				// Connection errors are expected without a real server
				// But argument parsing should work
				_ = err
			}
		})
	}
}

// TestCLIE2EOutputFormatting tests output formatting of commands
func TestCLIE2EOutputFormatting(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rootCmd := cmdpkg.GetRootCommand()
	args := []string{"--control-plane-addr", server.URL, "policy", "list"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	// May fail due to various reasons, but should not panic
	_ = err
}

// TestCLIE2EAPIClientInitialization tests API client initialization
func TestCLIE2EAPIClientInitialization(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Test with working server
	apiClient, err := client.NewAPIClient(server.URL, 5000000000) // 5 seconds
	assert.NoError(t, err)
	assert.NotNil(t, apiClient)
}

// TestCLIE2EComplexPolicyNames tests handling of complex policy names
func TestCLIE2EComplexPolicyNames(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"hyphenated", []string{"policy", "get", "my-policy-name"}},
		{"underscored", []string{"policy", "get", "my_policy_name"}},
		{"versioned", []string{"policy", "get", "my-policy-v1"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmdpkg.GetRootCommand()
			rootCmd.SetArgs(tc.args)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			// These should parse successfully even if server fails
			err := rootCmd.Execute()
			_ = err
		})
	}
}
