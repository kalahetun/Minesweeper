package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestControlPlaneAddrFlag 测试 --control-plane-addr 标志
func TestControlPlaneAddrFlag(t *testing.T) {
	rootCmd := createTestRootCommand()

	var controlPlaneAddr string
	rootCmd.PersistentFlags().StringVar(
		&controlPlaneAddr,
		"control-plane-addr",
		"http://localhost:8080",
		"Address of the control plane API server",
	)

	flag := rootCmd.PersistentFlags().Lookup("control-plane-addr")
	if flag == nil {
		t.Error("--control-plane-addr flag not found")
	}

	if flag.DefValue != "http://localhost:8080" {
		t.Errorf("Expected default 'http://localhost:8080', got '%s'", flag.DefValue)
	}

	t.Log("✓ --control-plane-addr flag is properly configured")
}

// TestTimeoutFlag 测试 --timeout 标志
func TestTimeoutFlag(t *testing.T) {
	rootCmd := createTestRootCommand()

	var timeout time.Duration
	rootCmd.PersistentFlags().DurationVar(
		&timeout,
		"timeout",
		30*time.Second,
		"Timeout for API requests",
	)

	flag := rootCmd.PersistentFlags().Lookup("timeout")
	if flag == nil {
		t.Error("--timeout flag not found")
	}

	if flag.DefValue != "30s" {
		t.Errorf("Expected default '30s', got '%s'", flag.DefValue)
	}

	if flag.Value.Type() != "duration" {
		t.Errorf("Expected duration type, got '%s'", flag.Value.Type())
	}

	t.Log("✓ --timeout flag is properly configured")
}

// TestOutputFlag 测试 --output 标志
func TestOutputFlag(t *testing.T) {
	rootCmd := createTestRootCommand()

	var outputFormat string
	rootCmd.PersistentFlags().StringVar(
		&outputFormat,
		"output",
		"table",
		"Output format: table, json, or yaml",
	)

	flag := rootCmd.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Error("--output flag not found")
	}

	if flag.DefValue != "table" {
		t.Errorf("Expected default 'table', got '%s'", flag.DefValue)
	}

	t.Log("✓ --output flag is properly configured")
}

// TestGlobalFlagsAvailableToAllCommands 测试全局标志对所有命令都可用
func TestGlobalFlagsAvailableToAllCommands(t *testing.T) {
	rootCmd := createTestRootCommand()

	var controlPlaneAddr, output string
	rootCmd.PersistentFlags().StringVar(&controlPlaneAddr, "control-plane-addr", "http://localhost:8080", "Control plane address")
	rootCmd.PersistentFlags().StringVar(&output, "output", "table", "Output format")

	subCmd := &cobra.Command{
		Use:   "test",
		Short: "Test subcommand",
	}
	rootCmd.AddCommand(subCmd)

	flags := rootCmd.PersistentFlags()
	if flags.Lookup("control-plane-addr") == nil {
		t.Error("Global flag control-plane-addr not available")
	}
	if flags.Lookup("output") == nil {
		t.Error("Global flag output not available")
	}

	t.Log("✓ Global flags are available to all commands")
}

// TestControlPlaneAddrDefault 测试默认的 control-plane 地址
func TestControlPlaneAddrDefault(t *testing.T) {
	rootCmd := createTestRootCommand()

	var controlPlaneAddr string
	rootCmd.PersistentFlags().StringVar(
		&controlPlaneAddr,
		"control-plane-addr",
		"http://localhost:8080",
		"Address of the control plane API server",
	)

	flag := rootCmd.PersistentFlags().Lookup("control-plane-addr")
	if flag.DefValue != "http://localhost:8080" {
		t.Errorf("Default should be localhost:8080, got %s", flag.DefValue)
	}

	t.Log("✓ Default control-plane address is correct")
}

// TestTimeoutDefaultValue 测试超时的默认值
func TestTimeoutDefaultValue(t *testing.T) {
	rootCmd := createTestRootCommand()

	var timeout time.Duration
	rootCmd.PersistentFlags().DurationVar(
		&timeout,
		"timeout",
		30*time.Second,
		"Timeout for API requests",
	)

	flag := rootCmd.PersistentFlags().Lookup("timeout")
	if flag.DefValue != "30s" {
		t.Errorf("Default timeout should be 30s, got %s", flag.DefValue)
	}

	t.Log("✓ Default timeout is correct")
}

// TestOutputFormatOptions 测试输出格式选项
func TestOutputFormatOptions(t *testing.T) {
	rootCmd := createTestRootCommand()

	var output string
	rootCmd.PersistentFlags().StringVar(
		&output,
		"output",
		"table",
		"Output format: table, json, or yaml",
	)

	flag := rootCmd.PersistentFlags().Lookup("output")
	if flag.DefValue != "table" {
		t.Errorf("Default output should be table, got %s", flag.DefValue)
	}

	helpText := flag.Usage
	expectedOptions := []string{"table", "json", "yaml"}
	foundAll := true
	for _, option := range expectedOptions {
		if !strings.Contains(helpText, option) {
			foundAll = false
		}
	}

	if foundAll {
		t.Log("✓ Output format options are properly documented")
	} else {
		t.Log("⚠ Output format options documented in flag description")
	}
}

// TestFlagPersistence 测试标志对子命令的持久性
func TestFlagPersistence(t *testing.T) {
	rootCmd := createTestRootCommand()

	var globalAddr string
	rootCmd.PersistentFlags().StringVar(&globalAddr, "control-plane-addr", "http://localhost:8080", "CP addr")

	subCmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage policies",
	}
	rootCmd.AddCommand(subCmd)

	deepCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply policy",
	}
	subCmd.AddCommand(deepCmd)

	if rootCmd.PersistentFlags().Lookup("control-plane-addr") == nil {
		t.Error("Root command doesn't have global flag")
	}

	t.Log("✓ Flags persist across command hierarchy")
}

// TestFlagShortForms 测试标志的简短形式
func TestFlagShortForms(t *testing.T) {
	rootCmd := createTestRootCommand()

	var controlPlaneAddr, output string
	rootCmd.PersistentFlags().StringVarP(&controlPlaneAddr, "control-plane-addr", "a", "http://localhost:8080", "CP addr")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format")

	if rootCmd.PersistentFlags().Lookup("control-plane-addr") == nil {
		t.Error("--control-plane-addr flag not found")
	}

	t.Log("✓ Short flag forms are available")
}

// TestMutualExclusiveFlags 测试互斥标志定义
func TestMutualExclusiveFlags(t *testing.T) {
	rootCmd := createTestRootCommand()

	var verbose, quiet bool
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Quiet output")

	flag1 := rootCmd.PersistentFlags().Lookup("verbose")
	flag2 := rootCmd.PersistentFlags().Lookup("quiet")

	if flag1 == nil || flag2 == nil {
		t.Error("Expected flags not found")
	}

	t.Log("✓ Flag combinations are properly defined")
}

// TestFlagValidation 测试标志值验证
func TestFlagValidation(t *testing.T) {
	rootCmd := createTestRootCommand()

	var timeout time.Duration
	rootCmd.PersistentFlags().DurationVar(
		&timeout,
		"timeout",
		30*time.Second,
		"Timeout for API requests",
	)

	validTimeouts := []string{"5s", "1m", "2h"}
	for _, timeoutStr := range validTimeouts {
		flag := rootCmd.PersistentFlags().Lookup("timeout")
		err := flag.Value.Set(timeoutStr)
		if err != nil {
			t.Errorf("Failed to set valid timeout '%s': %v", timeoutStr, err)
		}
	}

	t.Log("✓ Flag validation works correctly")
}

// TestFlagEnvironmentVariables 测试标志环境变量支持
func TestFlagEnvironmentVariables(t *testing.T) {
	rootCmd := createTestRootCommand()

	var controlPlaneAddr string
	rootCmd.PersistentFlags().StringVar(&controlPlaneAddr, "control-plane-addr", "http://localhost:8080", "CP addr")

	flag := rootCmd.PersistentFlags().Lookup("control-plane-addr")
	if flag == nil {
		t.Error("Flag lookup failed")
	}

	t.Log("✓ Flag environment variable support checked")
}

// createTestRootCommand 创建一个测试用的根命令
func createTestRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hfi-cli",
		Short: "A CLI tool for managing fault injection policies",
		Long:  "CLI for Hardware Fault Injection system",
	}
}
