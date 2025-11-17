package cmd

import (
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/vault"
	"golang.org/x/term"
)

var (
	unlockedVault *vault.Vault
	vaultKey      []byte
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault",
	Long:  `Unlock the vault by providing the master password.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if unlockedVault != nil {
			fmt.Println("Vault is already unlocked")
			return nil
		}

		if !localStore.Exists() {
			return fmt.Errorf("vault not found. Run 'vaultctl init' first")
		}

		// Prompt for master password
		fmt.Print("Enter master password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// Try to load from local first
		v, key, err := localStore.DecryptAndLoad(password)
		if err != nil {
			// Try loading from DynamoDB if local fails
			if dynamoStore != nil {
				ctx := cmd.Context()
				if ctx == nil {
					ctx = context.Background()
				}
				ev, err2 := dynamoStore.LoadVault(ctx)
				if err2 != nil {
					return fmt.Errorf("failed to unlock vault: %w (also failed to load from DynamoDB: %v)", err, err2)
				}
				// Decrypt from DynamoDB vault
				v, key, err = decryptVaultFromEncrypted(ev, password)
				if err != nil {
					return fmt.Errorf("failed to decrypt vault from DynamoDB: %w", err)
				}
			} else {
				return fmt.Errorf("failed to unlock vault: %w", err)
			}
		}

		unlockedVault = v
		vaultKey = key
		fmt.Println("Vault unlocked successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}

// ensureUnlocked ensures the vault is unlocked, prompting if necessary
func ensureUnlocked(cmd *cobra.Command) error {
	if unlockedVault != nil {
		return nil
	}
	return unlockCmd.RunE(cmd, nil)
}

