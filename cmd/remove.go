package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name_or_id>",
	Short: "Remove a password entry",
	Long:  `Remove a password entry from the vault by name or ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(cmd); err != nil {
			return err
		}

		if !unlockedVault.RemoveEntry(args[0]) {
			return fmt.Errorf("entry not found: %s", args[0])
		}

		// Save vault
		sync := !cmd.Flags().Changed("no-sync")
		if err := saveVault(cmd, sync); err != nil {
			return fmt.Errorf("failed to save vault: %w", err)
		}

		fmt.Printf("Entry '%s' removed successfully\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().Bool("no-sync", false, "Don't sync to DynamoDB")
}

