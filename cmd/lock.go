package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock the vault and clear session",
	Long:  `Lock the vault by clearing the session. You will need to unlock again to use the vault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Clear in-memory state
		unlockedVault = nil
		vaultKey = nil

		// Clear session
		if sessionMgr != nil {
			if err := sessionMgr.ClearSession(); err != nil {
				return fmt.Errorf("failed to clear session: %w", err)
			}
		}

		fmt.Println("Vault locked successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}

