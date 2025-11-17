package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/crypto"
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

		// Save session for future commands
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		if err := sessionMgr.SaveSession(ctx, key); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save session: %v\n", err)
		}
		
		// Zeroize master password from memory
		crypto.Zeroize(password)

		fmt.Println("Vault unlocked successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}

// ensureUnlocked ensures the vault is unlocked, prompting if necessary
func ensureUnlocked(cmd *cobra.Command) error {
	// Check if already unlocked in memory
	if unlockedVault != nil && vaultKey != nil {
		return nil
	}

	// Try to load from session
	if sessionMgr != nil {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		if key, err := sessionMgr.LoadSession(ctx); err == nil {
			// Session is valid, decrypt vault with the key
			ev, err := localStore.LoadEncryptedVault()
			if err != nil {
				// Try DynamoDB if local fails
				if dynamoStore != nil {
					ctx := cmd.Context()
					if ctx == nil {
						ctx = context.Background()
					}
					ev, err = dynamoStore.LoadVault(ctx)
					if err != nil {
						return fmt.Errorf("failed to load vault: %w", err)
					}
				} else {
					return fmt.Errorf("failed to load vault: %w", err)
				}
			}

			// Decrypt vault using the session key
			ciphertext, err := crypto.DecodeBase64(ev.Ciphertext)
			if err != nil {
				return fmt.Errorf("failed to decode ciphertext: %w", err)
			}

			nonce, err := crypto.DecodeBase64(ev.Nonce)
			if err != nil {
				return fmt.Errorf("failed to decode nonce: %w", err)
			}

			plaintext, err := crypto.Decrypt(ciphertext, nonce, key)
			if err != nil {
				// Session key might be invalid, clear session and prompt
				sessionMgr.ClearSession()
				return unlockCmd.RunE(cmd, nil)
			}

			v, err := vault.FromJSON(plaintext)
			if err != nil {
				return fmt.Errorf("failed to deserialize vault: %w", err)
			}

			unlockedVault = v
			vaultKey = key
			return nil
		}
		// Session expired or invalid, continue to prompt
	}

	// No valid session, prompt for password
	return unlockCmd.RunE(cmd, nil)
}

