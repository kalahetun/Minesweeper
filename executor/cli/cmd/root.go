package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"hfi-cli/client"

	"github.com/spf13/cobra"
)

var (
	// globalFlags holds the global CLI configuration
	globalFlags client.GlobalFlags
	
	// apiClient is the global API client instance
	apiClient client.IAPIClient
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hfi-cli",
	Short: "A CLI tool for managing fault injection policies",
	Long: `hfi-cli is a command line interface for the Hardware Fault Injection (HFI) system.
It allows you to manage fault injection policies that control how faults are injected
into your services through the Envoy proxy and WebAssembly plugin.

Examples:
  hfi-cli policy apply -f my-policy.yaml
  hfi-cli policy get
  hfi-cli policy delete my-policy`,
	
	// PersistentPreRunE is called before any subcommand runs
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the API client with global flags
		client, err := client.NewAPIClient(globalFlags.ControlPlaneAddr, globalFlags.Timeout)
		if err != nil {
			return fmt.Errorf("failed to initialize API client: %w", err)
		}
		
		// Store the client globally for subcommands to use
		apiClient = client
		
		// Optional: Perform a health check to validate the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := apiClient.HealthCheck(ctx); err != nil {
			// Don't fail initialization on health check failure, just warn
			fmt.Fprintf(os.Stderr, "Warning: Control plane health check failed: %v\n", err)
		}
		
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Define persistent flags that are available to all subcommands
	rootCmd.PersistentFlags().StringVar(
		&globalFlags.ControlPlaneAddr,
		"control-plane-addr",
		"http://localhost:8080",
		"Address of the control plane API server",
	)
	
	rootCmd.PersistentFlags().DurationVar(
		&globalFlags.Timeout,
		"timeout",
		30*time.Second,
		"Timeout for API requests to the control plane",
	)
	
	// Add subcommands
	rootCmd.AddCommand(policyCmd)
}

// GetAPIClient returns the global API client instance
// This function is used by subcommands to access the initialized client
func GetAPIClient() client.IAPIClient {
	return apiClient
}
