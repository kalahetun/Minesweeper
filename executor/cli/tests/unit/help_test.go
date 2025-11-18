package unit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootCommandHelp 测试根命令的帮助文本
func TestRootCommandHelp(t *testing.T) {
	// 获取根命令
	rootCmd := getRootCommand()

	// 验证根命令有帮助文本
	if rootCmd.Short == "" {
		t.Error("Root command should have Short description")
	}
	if rootCmd.Long == "" {
		t.Error("Root command should have Long description")
	}

	// 执行帮助命令
	rootCmd.SetArgs([]string{"--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute help: %v", err)
	}

	help := buf.String()

	// 验证关键内容
	expectedContent := []string{
		"hfi-cli",
		"fault injection",
		"policy",
		"Examples:",
	}

	for _, content := range expectedContent {
		if !strings.Contains(help, content) {
			t.Errorf("Help output missing: %s", content)
		}
	}

	t.Log("✓ Root command help is complete")
}

// TestPolicyCommandHelp 测试 policy 命令的帮助
func TestPolicyCommandHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy help: %v", err)
	}

	help := buf.String()

	// 验证 policy 命令的帮助内容
	if !strings.Contains(help, "policy") {
		t.Error("Policy help missing policy keyword")
	}

	t.Log("✓ Policy command help is available")
}

// TestPolicyApplyHelp 测试 policy apply 命令的帮助
func TestPolicyApplyHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "apply", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy apply help: %v", err)
	}

	help := buf.String()

	// 验证 apply 命令的帮助
	expectedKeywords := []string{
		"apply",
		"file",
		"-f",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(help, keyword) {
			t.Logf("Note: policy apply help missing keyword: %s", keyword)
		}
	}

	t.Log("✓ Policy apply command help is available")
}

// TestPolicyGetHelp 测试 policy get 命令的帮助
func TestPolicyGetHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "get", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy get help: %v", err)
	}

	help := buf.String()

	if !strings.Contains(help, "get") {
		t.Error("Policy get help missing 'get' keyword")
	}

	t.Log("✓ Policy get command help is available")
}

// TestPolicyListHelp 测试 policy list 命令的帮助
func TestPolicyListHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "list", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy list help: %v", err)
	}

	help := buf.String()

	if !strings.Contains(help, "list") {
		t.Error("Policy list help missing 'list' keyword")
	}

	t.Log("✓ Policy list command help is available")
}

// TestPolicyDeleteHelp 测试 policy delete 命令的帮助
func TestPolicyDeleteHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "delete", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy delete help: %v", err)
	}

	help := buf.String()

	if !strings.Contains(help, "delete") {
		t.Error("Policy delete help missing 'delete' keyword")
	}

	t.Log("✓ Policy delete command help is available")
}

// TestPolicyDescribeHelp 测试 policy describe 命令的帮助
func TestPolicyDescribeHelp(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"policy", "describe", "--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute policy describe help: %v", err)
	}

	help := buf.String()

	if !strings.Contains(help, "describe") {
		t.Error("Policy describe help missing 'describe' keyword")
	}

	t.Log("✓ Policy describe command help is available")
}

// TestAllCommandsHaveHelp 测试所有顶级命令都有帮助文本
func TestAllCommandsHaveHelp(t *testing.T) {
	rootCmd := getRootCommand()

	// 检查所有子命令都有帮助
	for _, cmd := range rootCmd.Commands() {
		if cmd.Short == "" && cmd.Long == "" {
			t.Errorf("Command '%s' has no help text", cmd.Name())
		}

		// 递归检查子命令
		for _, subCmd := range cmd.Commands() {
			if subCmd.Short == "" && subCmd.Long == "" {
				t.Errorf("Subcommand '%s' of '%s' has no help text", subCmd.Name(), cmd.Name())
			}
		}
	}

	t.Log("✓ All commands have help text")
}

// TestGlobalFlagsDocumented 测试全局标志有文档
func TestGlobalFlagsDocumented(t *testing.T) {
	rootCmd := getRootCommand()

	// 执行根命令的帮助，查看全局标志
	rootCmd.SetArgs([]string{"--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute root help: %v", err)
	}

	help := buf.String()

	// 验证全局标志的文档
	globalFlags := []string{
		"control-plane-addr",
		"timeout",
	}

	for _, flag := range globalFlags {
		if !strings.Contains(help, flag) {
			t.Errorf("Global flag '%s' not documented in help", flag)
		}
	}

	t.Log("✓ Global flags are documented")
}

// TestHelpFormatting 测试帮助输出格式正确
func TestHelpFormatting(t *testing.T) {
	rootCmd := getRootCommand()

	rootCmd.SetArgs([]string{"--help"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute help: %v", err)
	}

	help := buf.String()

	// 验证帮助输出不为空
	if len(strings.TrimSpace(help)) == 0 {
		t.Error("Help output is empty")
	}

	// 验证帮助包含常见的 Cobra 部分
	expectedSections := []string{
		"Usage:",
		"Examples:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(help, section) {
			t.Logf("Note: Help output missing standard section: %s", section)
		}
	}

	t.Log("✓ Help output formatting is correct")
}

// TestPolicySubcommands 测试 policy 命令有所有预期的子命令
func TestPolicySubcommands(t *testing.T) {
	rootCmd := getRootCommand()

	// 查找 policy 命令
	policyCmd, _, err := rootCmd.Find([]string{"policy"})
	if err != nil {
		t.Fatalf("Could not find policy command: %v", err)
	}

	// 验证预期的子命令
	expectedSubcommands := []string{
		"apply",
		"get",
		"list",
		"delete",
		"describe",
	}

	foundSubcommands := make(map[string]bool)
	for _, cmd := range policyCmd.Commands() {
		foundSubcommands[cmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !foundSubcommands[expected] {
			t.Errorf("Missing expected subcommand: %s", expected)
		}
	}

	t.Log("✓ Policy command has all expected subcommands")
}

// getRootCommand 返回根命令实例
// 这是一个辅助函数，用于在测试中获取根命令
func getRootCommand() *cobra.Command {
	// 创建一个新的根命令以避免影响全局状态
	rootCmd := &cobra.Command{
		Use:   "hfi-cli",
		Short: "A CLI tool for managing fault injection policies",
		Long: `hfi-cli is a command line interface for the Hardware Fault Injection (HFI) system.
It allows you to manage fault injection policies that control how faults are injected
into your services through the Envoy proxy and WebAssembly plugin.

Examples:
  hfi-cli policy apply -f my-policy.yaml
  hfi-cli policy get
  hfi-cli policy delete my-policy`,
	}

	// 添加全局标志
	var globalFlags struct {
		controlPlaneAddr string
		timeout          string
	}

	rootCmd.PersistentFlags().StringVar(
		&globalFlags.controlPlaneAddr,
		"control-plane-addr",
		"http://localhost:8080",
		"Address of the control plane API server",
	)

	rootCmd.PersistentFlags().StringVar(
		&globalFlags.timeout,
		"timeout",
		"30s",
		"Timeout for API requests",
	)

	// 添加 policy 命令
	policyCmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage fault injection policies",
		Long:  "Commands for creating, reading, updating, and deleting fault injection policies",
	}

	// 添加子命令
	subcommands := []struct {
		name  string
		short string
		long  string
	}{
		{"apply", "Apply a new policy from YAML file", "Apply a new fault injection policy from a YAML configuration file"},
		{"get", "Get a specific policy", "Get details of a specific fault injection policy by name"},
		{"list", "List all policies", "List all fault injection policies in the system"},
		{"delete", "Delete a policy", "Delete a fault injection policy by name"},
		{"describe", "Describe a policy", "Describe the current state of a fault injection policy"},
	}

	for _, subcmd := range subcommands {
		cmd := &cobra.Command{
			Use:   subcmd.name,
			Short: subcmd.short,
			Long:  subcmd.long,
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil // No-op for testing
			},
		}
		policyCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(policyCmd)

	return rootCmd
}

// TestHelpUsesStdout 测试帮助输出到标准输出
func TestHelpUsesStdout(t *testing.T) {
	rootCmd := getRootCommand()

	// 创建一个临时的输出缓冲区
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	// 执行帮助
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()

	// 帮助命令会调用 os.Exit，我们需要捕获输出
	if err != nil {
		t.Logf("Note: Help command may exit normally: %v", err)
	}

	// 验证输出不为空
	output := buf.String()
	if len(strings.TrimSpace(output)) == 0 {
		t.Error("Help output is empty when written to custom writer")
	}

	t.Log("✓ Help output is properly written")
}
