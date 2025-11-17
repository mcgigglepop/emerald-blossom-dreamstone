package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/crypto"
	"github.com/vaultctl/vaultctl/internal/storage"
	"github.com/vaultctl/vaultctl/internal/vault"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vault",
	Long:  `Initialize a new encrypted vault with a master password.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if localStore.Exists() {
			return fmt.Errorf("vault already exists at %s. Use 'vaultctl unlock' to access it", cfg.VaultPath)
		}

		// Prompt for master password
		fmt.Print("Enter master password: ")
		password1, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		fmt.Print("Confirm master password: ")
		password2, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		if !crypto.ConstantTimeCompare(password1, password2) {
			return fmt.Errorf("passwords do not match")
		}

		// Generate salt and vault key
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}

		vaultKey, err := crypto.GenerateVaultKey()
		if err != nil {
			return fmt.Errorf("failed to generate vault key: %w", err)
		}

		// Derive master key
		kdfParams := crypto.DefaultKDFParams()
		masterKey := crypto.DeriveMasterKey(password1, salt, kdfParams)

		// Encrypt vault key
		encVaultKey, vaultKeyNonce, err := crypto.EncryptVaultKey(vaultKey, masterKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt vault key: %w", err)
		}

		// Create empty vault
		v := vault.NewVault()

		// Encrypt vault
		plaintext, err := v.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize vault: %w", err)
		}

		ciphertext, nonce, err := crypto.Encrypt(plaintext, vaultKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt vault: %w", err)
		}

		// Create encrypted vault structure
		ev := &storage.EncryptedVault{
			SchemaVersion: vault.SchemaVersion,
			VaultID:       v.VaultID,
			SaltMaster:    crypto.EncodeBase64(salt),
			EncVaultKey:   crypto.EncodeBase64(encVaultKey),
			VaultKeyNonce: crypto.EncodeBase64(vaultKeyNonce),
			KDFParams: storage.KDFParams{
				Algo:       kdfParams.Algo,
				Memory:     kdfParams.Memory,
				Iterations: kdfParams.Iterations,
				Parallelism: kdfParams.Parallelism,
			},
			Cipher:     "xchacha20poly1305",
			Ciphertext: crypto.EncodeBase64(ciphertext),
			Nonce:      crypto.EncodeBase64(nonce),
			Version:    1,
		}
		ev.SetModifiedAt(time.Now())

		// Save locally
		if err := localStore.SaveEncryptedVault(ev); err != nil {
			return fmt.Errorf("failed to save vault locally: %w", err)
		}

		// Save to DynamoDB if available
		if dynamoStore != nil {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = cmd.Root().Context()
			}
			if err := dynamoStore.SaveVault(ctx, ev, 0); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save to DynamoDB: %v\n", err)
			} else {
				fmt.Println("Vault initialized and synced to DynamoDB")
			}
		} else {
			fmt.Println("Vault initialized locally")
		}

		// Save config
		if err := cfg.SaveConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

