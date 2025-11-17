package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/config"
	"github.com/vaultctl/vaultctl/internal/storage"
)

var (
	cfg        *config.Config
	localStore *storage.LocalStorage
	dynamoStore *storage.DynamoDBStorage
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vaultctl",
	Short: "A zero-knowledge CLI password manager",
	Long: `vaultctl is a CLI password manager with client-side encryption.
All encryption and decryption happens locally. The server (DynamoDB) only
stores encrypted blobs and never sees your master password or decrypted data.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	localStore = storage.NewLocalStorage(cfg.VaultPath)

	// Try to initialize DynamoDB storage, but don't fail if it's not configured
	dynamoStore, err = storage.NewDynamoDBStorage(cfg.TableName, cfg.UserID)
	if err != nil {
		// Don't fail if DynamoDB isn't configured, just log
		fmt.Fprintf(os.Stderr, "Warning: DynamoDB not available: %v\n", err)
		dynamoStore = nil
	}

	return rootCmd.Execute()
}

func init() {
	// Flags will be set after config is loaded in Execute()
}

