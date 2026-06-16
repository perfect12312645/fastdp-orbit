package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "orbitctl",
	Short: "Orbit - Heterogeneous Computing Platform",
	Long:  `A platform for managing heterogeneous computing resources including Kubernetes clusters.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")

	// Add subcommands
	rootCmd.AddCommand(machineCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(workflowCmd)
	rootCmd.AddCommand(versionCmd)
}
