package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/crypto"
	"golang.org/x/term"
)

var rotateMasterCmd = &cobra.Command{
	Use:   "rotate-master",
	Short: "Change the master password",
	Long:  `Change the master password by re-encrypting the vault key with a new master key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !localStore.Exists() {
			return fmt.Errorf("vault not found. Run 'vaultctl init' first")
		}

		// Load encrypted vault
		ev, err := localStore.LoadEncryptedVault()
		if err != nil {
			return fmt.Errorf("failed to load vault: %w", err)
		}

		// Prompt for current master password
		fmt.Print("Enter current master password: ")
		currentPassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// Decrypt vault key with current password
		salt, err := crypto.DecodeBase64(ev.SaltMaster)
		if err != nil {
			return fmt.Errorf("failed to decode salt: %w", err)
		}

		encVaultKey, err := crypto.DecodeBase64(ev.EncVaultKey)
		if err != nil {
			return fmt.Errorf("failed to decode encrypted vault key: %w", err)
		}

		kdfParams := crypto.KDFParams{
			Algo:       ev.KDFParams.Algo,
			Memory:     ev.KDFParams.Memory,
			Iterations: ev.KDFParams.Iterations,
			Parallelism: ev.KDFParams.Parallelism,
		}
		currentMasterKey := crypto.DeriveMasterKey(currentPassword, salt, kdfParams)

		var vaultKeyNonce []byte
		if ev.VaultKeyNonce != "" {
			vaultKeyNonce, err = crypto.DecodeBase64(ev.VaultKeyNonce)
			if err != nil {
				return fmt.Errorf("failed to decode vault key nonce: %w", err)
			}
		} else {
			// Backward compatibility
			vaultKeyNonce, err = crypto.DecodeBase64(ev.Nonce)
			if err != nil {
				return fmt.Errorf("failed to decode nonce: %w", err)
			}
		}

		vaultKey, err := crypto.DecryptVaultKey(encVaultKey, vaultKeyNonce, currentMasterKey)
		if err != nil {
			crypto.Zeroize(currentPassword)
			return fmt.Errorf("failed to decrypt vault key: %w", err)
		}
		
		// Zeroize current password and master key after use
		crypto.Zeroize(currentPassword)
		crypto.Zeroize(currentMasterKey)

		// Prompt for new master password
		fmt.Print("Enter new master password: ")
		newPassword1, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		fmt.Print("Confirm new master password: ")
		newPassword2, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		if !crypto.ConstantTimeCompare(newPassword1, newPassword2) {
			crypto.Zeroize(newPassword1)
			crypto.Zeroize(newPassword2)
			return fmt.Errorf("passwords do not match")
		}

		// Generate new salt
		newSalt, err := crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}

		// Derive new master key
		newMasterKey := crypto.DeriveMasterKey(newPassword1, newSalt, kdfParams)

		// Re-encrypt vault key with new master key
		newEncVaultKey, newNonceVK, err := crypto.EncryptVaultKey(vaultKey, newMasterKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt vault key: %w", err)
		}

		// Update encrypted vault
		ev.SaltMaster = crypto.EncodeBase64(newSalt)
		ev.EncVaultKey = crypto.EncodeBase64(newEncVaultKey)
		ev.VaultKeyNonce = crypto.EncodeBase64(newNonceVK)
		ev.SetModifiedAt(time.Now())
		ev.Version++

		// Save locally
		if err := localStore.SaveEncryptedVault(ev); err != nil {
			return fmt.Errorf("failed to save vault: %w", err)
		}

		// Save to DynamoDB if available
		if dynamoStore != nil {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if err := dynamoStore.SaveVault(ctx, ev, ev.Version-1); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save to DynamoDB: %v\n", err)
			}
		}

		// Zeroize all passwords and keys from memory
		crypto.Zeroize(newPassword1)
		crypto.Zeroize(newPassword2)
		crypto.Zeroize(newMasterKey)
		crypto.Zeroize(vaultKey)

		fmt.Println("Master password rotated successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rotateMasterCmd)
}

