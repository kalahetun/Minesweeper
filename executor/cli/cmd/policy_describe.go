package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"hfi-cli/client"
	"hfi-cli/types"
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   "describe POLICY_NAME",
	Short: "Show detailed information about a fault injection policy",
	Long: `Describe displays detailed information about a specific fault injection policy,
including all rules, match conditions, and fault configurations.

The command outputs the complete policy definition in the specified format
(table, YAML, or JSON).

Examples:
  # Describe a policy (table format)
  hfi-cli policy describe my-policy
  
  # Describe a policy in YAML format
  hfi-cli policy describe my-policy -o yaml
  
  # Describe a policy in JSON format
  hfi-cli policy describe my-policy -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		policyName := args[0]

		// Get the API client
		apiClient := GetAPIClient()
		if apiClient == nil {
			return fmt.Errorf("API client not initialized")
		}

		// Retrieve the policy
		policy, err := apiClient.GetPolicyByName(ctx, policyName)
		if err != nil {
			if apiErr, ok := err.(*client.APIError); ok && apiErr.ErrCode == "NotFound" {
				return fmt.Errorf("policy %q not found", policyName)
			}
			return fmt.Errorf("failed to get policy: %w", err)
		}

		return outputPolicyDetailed(policy)
	},
}

// outputPolicyDetailed outputs detailed information about a policy
func outputPolicyDetailed(policy *types.FaultInjectionPolicy) error {
	switch outputFormat {
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(policy)
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(policy)
	case "table":
		return outputPolicyDetailedTable(policy)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// outputPolicyDetailedTable outputs a policy in detailed table format
func outputPolicyDetailedTable(policy *types.FaultInjectionPolicy) error {
	fmt.Printf("Name: %s\n", policy.Metadata.Name)
	fmt.Printf("Total Rules: %d\n\n", len(policy.Spec.Rules))

	for i, rule := range policy.Spec.Rules {
		fmt.Printf("Rule %d:\n", i+1)
		fmt.Printf("  Match Conditions:\n")

		if rule.Match.Method != nil {
			fmt.Printf("    Method:\n")
			if rule.Match.Method.Exact != "" {
				fmt.Printf("      Exact: %s\n", rule.Match.Method.Exact)
			}
			if rule.Match.Method.Prefix != "" {
				fmt.Printf("      Prefix: %s\n", rule.Match.Method.Prefix)
			}
			if rule.Match.Method.Regex != "" {
				fmt.Printf("      Regex: %s\n", rule.Match.Method.Regex)
			}
		}

		if rule.Match.Path != nil {
			fmt.Printf("    Path:\n")
			if rule.Match.Path.Exact != "" {
				fmt.Printf("      Exact: %s\n", rule.Match.Path.Exact)
			}
			if rule.Match.Path.Prefix != "" {
				fmt.Printf("      Prefix: %s\n", rule.Match.Path.Prefix)
			}
			if rule.Match.Path.Regex != "" {
				fmt.Printf("      Regex: %s\n", rule.Match.Path.Regex)
			}
		}

		if len(rule.Match.Headers) > 0 {
			fmt.Printf("    Headers:\n")
			for _, header := range rule.Match.Headers {
				fmt.Printf("      %s:\n", header.Name)
				if header.Exact != "" {
					fmt.Printf("        Exact: %s\n", header.Exact)
				}
				if header.Prefix != "" {
					fmt.Printf("        Prefix: %s\n", header.Prefix)
				}
				if header.Regex != "" {
					fmt.Printf("        Regex: %s\n", header.Regex)
				}
			}
		}

		fmt.Printf("  Fault Actions:\n")
		fmt.Printf("    Percentage: %d%%\n", rule.Fault.Percentage)
		fmt.Printf("    Start Delay: %d ms\n", rule.Fault.StartDelayMs)
		fmt.Printf("    Duration: %d seconds\n", rule.Fault.DurationSeconds)

		if rule.Fault.Delay != nil {
			fmt.Printf("    Delay: %d ms\n", rule.Fault.Delay.FixedDelayMs)
		}

		if rule.Fault.Abort != nil {
			fmt.Printf("    Abort HTTP Status: %d\n", rule.Fault.Abort.HTTPStatus)
		}

		fmt.Printf("\n")
	}

	return nil
}

func init() {
	describeCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, yaml, json")
}
