package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync vault with DynamoDB",
	Long:  `Sync the local vault with the remote vault in DynamoDB.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dynamoStore == nil {
			return fmt.Errorf("DynamoDB not configured")
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Load local encrypted vault
		localEV, err := localStore.LoadEncryptedVault()
		if err != nil {
			return fmt.Errorf("failed to load local vault: %w", err)
		}

		// Sync with remote
		syncedEV, err := dynamoStore.SyncVault(ctx, localEV)
		if err != nil {
			return fmt.Errorf("failed to sync vault: %w", err)
		}

		// If remote was newer, we need to reload the vault
		if syncedEV.Version > localEV.Version {
			fmt.Println("Remote vault is newer. Please unlock to reload.")
			// Reset unlocked state to force re-unlock
			unlockedVault = nil
			vaultKey = nil
		} else {
			// Save synced vault locally
			if err := localStore.SaveEncryptedVault(syncedEV); err != nil {
				return fmt.Errorf("failed to save synced vault: %w", err)
			}
			fmt.Printf("Vault synced successfully (version %d)\n", syncedEV.Version)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

