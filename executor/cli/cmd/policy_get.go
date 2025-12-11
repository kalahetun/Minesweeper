package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"hfi-cli/client"
	"hfi-cli/types"
)

var (
	outputFormat string
)

var getCmd = &cobra.Command{
	Use:   "get [POLICY_NAME]",
	Short: "Display one or many fault injection policies",
	Long: `Get displays fault injection policies.

If no policy name is specified, all policies will be listed.
If a policy name is specified, only that policy will be displayed.

Examples:
  # List all policies
  hfi-cli policy get

  # Get a specific policy
  hfi-cli policy get my-policy
  
  # Get policies in YAML format
  hfi-cli policy get -o yaml
  
  # Get a specific policy in JSON format
  hfi-cli policy get my-policy -o json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get the API client from the global flags
		apiClient := GetAPIClient()
		if apiClient == nil {
			return fmt.Errorf("API client not initialized")
		}

		// Determine if we're getting a specific policy or listing all
		if len(args) == 1 {
			// Get specific policy
			policyName := args[0]
			policy, err := apiClient.GetPolicyByName(ctx, policyName)
			if err != nil {
				if apiErr, ok := err.(*client.APIError); ok && apiErr.ErrCode == "NotFound" {
					return fmt.Errorf("policy %q not found", policyName)
				}
				return fmt.Errorf("failed to get policy: %w", err)
			}

			return outputPolicy(policy)
		} else {
			// List all policies
			policies, err := apiClient.ListPolicies(ctx)
			if err != nil {
				return fmt.Errorf("failed to list policies: %w", err)
			}

			return outputPolicies(policies)
		}
	},
}

func init() {
	getCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, yaml, json")
}

// outputPolicy formats and outputs a single policy
func outputPolicy(policy *types.FaultInjectionPolicy) error {
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
		return outputPolicyTable([]*types.FaultInjectionPolicy{policy})
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// outputPolicies formats and outputs multiple policies
func outputPolicies(policies []*types.FaultInjectionPolicy) error {
	switch outputFormat {
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(policies)
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(policies)
	case "table":
		return outputPolicyTable(policies)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// outputPolicyTable outputs policies in a human-readable table format
func outputPolicyTable(policies []*types.FaultInjectionPolicy) error {
	if len(policies) == 0 {
		fmt.Println("No fault injection policies found.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"NAME", "RULES", "MATCHES", "FAULTS"})

	for _, policy := range policies {
		name := policy.Metadata.Name
		rulesCount := strconv.Itoa(len(policy.Spec.Rules))

		// Summarize matches and faults
		matches := summarizeMatches(policy.Spec.Rules)
		faults := summarizeFaults(policy.Spec.Rules)

		table.Append([]string{name, rulesCount, matches, faults})
	}

	return table.Render()
}

// summarizeMatches creates a summary of match conditions
func summarizeMatches(rules []types.Rule) string {
	var matches []string

	for _, rule := range rules {
		var ruleSummary []string

		if rule.Match.Method != nil {
			if rule.Match.Method.Exact != "" {
				ruleSummary = append(ruleSummary, fmt.Sprintf("method=%s", rule.Match.Method.Exact))
			}
		}

		if rule.Match.Path != nil {
			if rule.Match.Path.Exact != "" {
				ruleSummary = append(ruleSummary, fmt.Sprintf("path=%s", rule.Match.Path.Exact))
			} else if rule.Match.Path.Prefix != "" {
				ruleSummary = append(ruleSummary, fmt.Sprintf("path=%s*", rule.Match.Path.Prefix))
			} else if rule.Match.Path.Regex != "" {
				ruleSummary = append(ruleSummary, fmt.Sprintf("path~%s", rule.Match.Path.Regex))
			}
		}

		if len(rule.Match.Headers) > 0 {
			ruleSummary = append(ruleSummary, fmt.Sprintf("headers(%d)", len(rule.Match.Headers)))
		}

		if len(ruleSummary) > 0 {
			matches = append(matches, strings.Join(ruleSummary, ","))
		}
	}

	if len(matches) == 0 {
		return "any"
	}

	// If there are multiple rules, show first rule + count
	if len(matches) > 1 {
		return fmt.Sprintf("%s (+%d more)", matches[0], len(matches)-1)
	}

	return matches[0]
}

// summarizeFaults creates a summary of fault types
func summarizeFaults(rules []types.Rule) string {
	var faults []string

	for _, rule := range rules {
		var ruleFaults []string

		if rule.Fault.Delay != nil {
			ruleFaults = append(ruleFaults, fmt.Sprintf("delay(%dms)", rule.Fault.Delay.FixedDelayMs))
		}

		if rule.Fault.Abort != nil {
			ruleFaults = append(ruleFaults, fmt.Sprintf("abort(%d)", rule.Fault.Abort.HTTPStatus))
		}

		// Add percentage info
		if len(ruleFaults) > 0 && rule.Fault.Percentage > 0 {
			percentage := fmt.Sprintf("%d%%", rule.Fault.Percentage)
			for i := range ruleFaults {
				ruleFaults[i] = fmt.Sprintf("%s@%s", ruleFaults[i], percentage)
			}
		}

		faults = append(faults, ruleFaults...)
	}

	if len(faults) == 0 {
		return "none"
	}

	// If there are multiple faults, show first + count
	if len(faults) > 1 {
		return fmt.Sprintf("%s (+%d more)", faults[0], len(faults)-1)
	}

	return faults[0]
}
