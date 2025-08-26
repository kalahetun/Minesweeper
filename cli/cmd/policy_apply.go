package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"hfi-cli/types"
)

var (
	// filename holds the path to the policy file
	filename string
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Create or update a fault injection policy from a file",
	Long: `Apply creates or updates a fault injection policy by reading the configuration
from a YAML or JSON file. The command is idempotent - you can run it multiple times
with the same file and it will create the policy if it doesn't exist or update it
if it does.

The policy file should contain a complete FaultInjectionPolicy specification
including metadata (name) and spec (rules).

Examples:
  hfi-cli policy apply -f my-policy.yaml
  hfi-cli policy apply --filename policy.json`,
	
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if filename == "" {
			return fmt.Errorf("-f, --filename is required")
		}
		
		// Read the policy file
		fileContent, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}
		
		// Parse the policy file (supports both YAML and JSON)
		var policy types.FaultInjectionPolicy
		if err := yaml.Unmarshal(fileContent, &policy); err != nil {
			return fmt.Errorf("failed to parse policy file: %w", err)
		}
		
		// Validate the policy
		if err := validatePolicy(&policy); err != nil {
			return fmt.Errorf("invalid policy: %w", err)
		}
		
		// Get the API client from root command
		apiClient := GetAPIClient()
		if apiClient == nil {
			return fmt.Errorf("API client not initialized")
		}
		
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Apply the policy
		if err := apiClient.CreateOrUpdatePolicy(ctx, &policy); err != nil {
			return fmt.Errorf("failed to apply policy: %w", err)
		}
		
		// Print success message in kubectl-style format
		fmt.Printf("faultinjectionpolicy.hfi.dev \"%s\" applied\n", policy.Metadata.Name)
		
		return nil
	},
}

// validatePolicy performs basic validation on the policy
func validatePolicy(policy *types.FaultInjectionPolicy) error {
	if policy.Metadata.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.Spec == nil {
		return fmt.Errorf("policy spec is required")
	}

	// Basic spec validation - check if spec has rules
	if specMap, ok := policy.Spec.(map[string]interface{}); ok {
		if rules, exists := specMap["rules"]; exists {
			if rulesSlice, ok := rules.([]interface{}); ok && len(rulesSlice) == 0 {
				return fmt.Errorf("at least one rule is required")
			}
		} else {
			return fmt.Errorf("spec must contain rules")
		}
	} else {
		return fmt.Errorf("spec must be an object")
	}

	return nil
}

func init() {
	// Add flags
	applyCmd.Flags().StringVarP(&filename, "filename", "f", "", "Path to the policy file (YAML or JSON)")
	
	// Make the filename flag required
	applyCmd.MarkFlagRequired("filename")
}
