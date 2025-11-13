package cmd

import (
	"github.com/spf13/cobra"
)

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage fault injection policies",
	Long: `The policy command is a container for all policy-related operations.
It provides subcommands to create, read, update, and delete fault injection policies.

Available subcommands:
  apply   - Create or update a policy from a file
  get     - Display one or many policies  
  delete  - Delete a policy by name

Examples:
  hfi-cli policy apply -f my-policy.yaml
  hfi-cli policy get
  hfi-cli policy get my-policy
  hfi-cli policy delete my-policy`,

	// This command doesn't have its own logic - it's just a container
	// Users must specify a subcommand
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add subcommands
	policyCmd.AddCommand(applyCmd)
	policyCmd.AddCommand(getCmd)
	policyCmd.AddCommand(listCmd)
	policyCmd.AddCommand(deleteCmd)
	policyCmd.AddCommand(describeCmd)
}
