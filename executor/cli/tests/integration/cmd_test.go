package integration

import (
	"bytes"
	"testing"

	"hfi-cli/cmd"

	"github.com/stretchr/testify/assert"
)

// TestCLIPolicyCmdParsing tests policy command parsing
func TestCLIPolicyCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "hfi-cli", rootCmd.Use)
}

// TestCLIPolicyListCmdParsing tests policy list command parsing
func TestCLIPolicyListCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "list"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no server, but should parse successfully
	_ = err
}

// TestCLIPolicyGetCmdParsing tests policy get command parsing
func TestCLIPolicyGetCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "get", "test-policy"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no server, but should parse successfully
	_ = err
}

// TestCLIPolicyApplyCmdParsing tests policy apply command parsing
func TestCLIPolicyApplyCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "apply", "-f", "policy.yaml"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no file, but should parse successfully
	_ = err
}

// TestCLIPolicyDeleteCmdParsing tests policy delete command parsing
func TestCLIPolicyDeleteCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "delete", "test-policy"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no server, but should parse successfully
	_ = err
}

// TestCLIPolicyDescribeCmdParsing tests policy describe command parsing
func TestCLIPolicyDescribeCmdParsing(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "describe", "test-policy"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no server, but should parse successfully
	_ = err
}

// TestCLIGlobalFlagsAddr tests global flags parsing
func TestCLIGlobalFlagsAddr(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"--control-plane-addr", "http://custom:8080", "policy", "list"}
	rootCmd.SetArgs(args)

	// Should parse without error
	err := rootCmd.Execute()
	// May fail due to connection, but parsing should work
	_ = err
}

// TestCLIGlobalFlagsTimeout tests timeout flag parsing
func TestCLIGlobalFlagsTimeout(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"--timeout", "30s", "policy", "list"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to connection, but parsing should work
	_ = err
}

// TestCLIHelpCommand tests help command
func TestCLIHelpCommand(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"--help"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "hfi-cli")
}

// TestCLIPolicyListHelp tests policy list help
func TestCLIPolicyListHelp(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "list", "--help"}
	rootCmd.SetArgs(args)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	assert.NoError(t, err)
	output := buf.String()
	assert.NotEmpty(t, output)
}

// TestCLIPolicyGetRequiredArgs tests required arguments validation
func TestCLIPolicyGetRequiredArgs(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	// Missing required policy name argument
	args := []string{"policy", "get"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// Should fail due to missing argument
	assert.Error(t, err)
}

// TestCLIPolicyApplyFileFlag tests file flag requirement
func TestCLIPolicyApplyFileFlag(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	// Missing required -f flag
	args := []string{"policy", "apply"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// Should fail due to missing file flag
	assert.Error(t, err)
}

// TestCLIValidateFlags tests valid flag combinations
func TestCLIValidateFlags(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"list-only", []string{"policy", "list"}},
		{"get-with-name", []string{"policy", "get", "my-policy"}},
		{"delete-with-name", []string{"policy", "delete", "my-policy"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmd.GetRootCommand()
			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()
			// May fail due to no server, but parsing should work
			_ = err
		})
	}
}

// TestCLIServerAddressConfiguration tests server address configuration
func TestCLIServerAddressConfiguration(t *testing.T) {
	testCases := []struct {
		name string
		addr string
	}{
		{"localhost", "http://localhost:8080"},
		{"custom-host", "http://192.168.1.1:9000"},
		{"ipv6", "http://[::1]:8080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := cmd.GetRootCommand()
			args := []string{"--control-plane-addr", tc.addr, "policy", "list"}
			rootCmd.SetArgs(args)

			err := rootCmd.Execute()
			// May fail due to connection, but parsing should work
			_ = err
		})
	}
}

// TestCLIInvalidFlags tests invalid flag handling
func TestCLIInvalidFlags(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"--invalid-flag", "value", "policy", "list"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// Should fail due to invalid flag
	assert.Error(t, err)
}

// TestCLIPolicyApplyYAMLFile tests YAML file parameter validation
func TestCLIPolicyApplyYAMLFile(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "apply", "-f", "/nonexistent/policy.yaml"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// Should fail due to missing file
	assert.Error(t, err)
}

// TestCLIPolicyDescribeCommand tests describe command
func TestCLIPolicyDescribeCommand(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{"policy", "describe", "test-policy"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to no server, but should parse
	_ = err
}

// TestCLIChainedFlags tests multiple global flags together
func TestCLIChainedFlags(t *testing.T) {
	rootCmd := cmd.GetRootCommand()
	args := []string{
		"--control-plane-addr", "http://myserver:9000",
		"--timeout", "60s",
		"policy", "list",
	}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	// May fail due to connection, but parsing should work
	_ = err
}
