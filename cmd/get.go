package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <name_or_id>",
	Short: "Get a password entry",
	Long:  `Get and display a password entry by name or ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(cmd); err != nil {
			return err
		}

		entry := unlockedVault.GetEntry(args[0])
		if entry == nil {
			return fmt.Errorf("entry not found: %s", args[0])
		}

		fmt.Printf("Name: %s\n", entry.Name)
		fmt.Printf("Username: %s\n", entry.Username)
		fmt.Printf("Password: %s\n", entry.Password)
		if entry.URL != "" {
			fmt.Printf("URL: %s\n", entry.URL)
		}
		if entry.Notes != "" {
			fmt.Printf("Notes: %s\n", entry.Notes)
		}
		if len(entry.BackupCodes) > 0 {
			fmt.Printf("Backup Codes:\n")
			for i, code := range entry.BackupCodes {
				fmt.Printf("  %d. %s\n", i+1, code)
			}
		}
		fmt.Printf("Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

