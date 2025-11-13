package cmd

import (
	"github.com/spf13/cobra"
)

// listCmd represents the list command (alias for get)
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all fault injection policies",
	Long: `List displays all fault injection policies in the system.

This is an alias for 'policy get' without a policy name.

Examples:
  # List all policies in table format
  hfi-cli policy list
  
  # List policies in YAML format
  hfi-cli policy list -o yaml
  
  # List policies in JSON format
  hfi-cli policy list -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delegate to getCmd with no policy name
		return getCmd.RunE(cmd, []string{})
	},
}

func init() {
	listCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, yaml, json")
}
