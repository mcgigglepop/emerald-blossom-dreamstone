package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [output_path]",
	Short: "Create a backup of the vault",
	Long:  `Create an encrypted backup of the vault.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !localStore.Exists() {
			return fmt.Errorf("vault not found. Run 'vaultctl init' first")
		}

		// Load encrypted vault
		ev, err := localStore.LoadEncryptedVault()
		if err != nil {
			return fmt.Errorf("failed to load vault: %w", err)
		}

		// Determine output path
		var outputPath string
		if len(args) > 0 {
			outputPath = args[0]
		} else {
			homeDir, _ := os.UserHomeDir()
			backupDir := filepath.Join(homeDir, ".vaultctl", "backups")
			if err := os.MkdirAll(backupDir, 0700); err != nil {
				return fmt.Errorf("failed to create backup directory: %w", err)
			}
			timestamp := time.Now().Format("2006-01-02T15-04-05Z")
			outputPath = filepath.Join(backupDir, fmt.Sprintf("vault-%s.enc", timestamp))
		}

		// Write backup
		data, err := ev.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize vault: %w", err)
		}

		if err := os.WriteFile(outputPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write backup: %w", err)
		}

		fmt.Printf("Backup created at: %s\n", outputPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}

