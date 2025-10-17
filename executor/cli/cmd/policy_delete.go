package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"hfi-cli/client"
)

var deleteCmd = &cobra.Command{
	Use:   "delete POLICY_NAME",
	Short: "Delete a fault injection policy",
	Long: `Delete removes a fault injection policy by name.

The policy will be immediately removed from the control plane and all
connected Envoy proxies will stop applying the fault injection rules
defined in that policy.

Examples:
  # Delete a policy
  hfi-cli policy delete my-policy`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		policyName := args[0]

		// Get the API client from the global flags
		apiClient := GetAPIClient()
		if apiClient == nil {
			return fmt.Errorf("API client not initialized")
		}

		// Delete the policy
		if err := apiClient.DeletePolicy(ctx, policyName); err != nil {
			if apiErr, ok := err.(*client.APIError); ok && apiErr.ErrCode == "NotFound" {
				return fmt.Errorf("policy %q not found", policyName)
			}
			return fmt.Errorf("failed to delete policy: %w", err)
		}

		// Print success message (similar to kubectl format)
		fmt.Printf("faultinjectionpolicy.hfi.dev %q deleted\n", policyName)
		return nil
	},
}
