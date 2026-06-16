package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Manage machines",
}

var machineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all machines",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement list machines
		fmt.Println("Listing machines...")
		return nil
	},
}

var machineAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement add machine
		fmt.Println("Adding machine...")
		return nil
	},
}

var machineRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement remove machine
		fmt.Println("Removing machine...")
		return nil
	},
}

var machineExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute command on machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement exec command
		fmt.Println("Executing command...")
		return nil
	},
}

func init() {
	machineAddCmd.Flags().StringP("name", "n", "", "machine name")
	machineAddCmd.Flags().StringP("ip", "i", "", "machine IP")
	machineAddCmd.Flags().IntP("port", "p", 22, "SSH port")
	machineAddCmd.Flags().StringP("user", "u", "", "username")

	machineExecCmd.Flags().StringP("command", "c", "", "command to execute")

	machineCmd.AddCommand(machineListCmd)
	machineCmd.AddCommand(machineAddCmd)
	machineCmd.AddCommand(machineRemoveCmd)
	machineCmd.AddCommand(machineExecCmd)
}
