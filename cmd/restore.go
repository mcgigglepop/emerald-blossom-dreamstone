package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/storage"
)

var restoreCmd = &cobra.Command{
	Use:   "restore [backup_path]",
	Short: "Restore vault from a backup",
	Long: `Restore your vault from an encrypted backup file.
If no backup path is provided, lists available backups for selection.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var backupPath string

		if len(args) > 0 {
			// Backup path provided directly
			backupPath = args[0]
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				return fmt.Errorf("backup file not found: %s", backupPath)
			}
		} else {
			// List and select from available backups
			homeDir, _ := os.UserHomeDir()
			backupDir := filepath.Join(homeDir, ".vaultctl", "backups")

			// Check if backup directory exists
			if _, err := os.Stat(backupDir); os.IsNotExist(err) {
				return fmt.Errorf("no backup directory found at %s. Create a backup first with 'vaultctl backup'", backupDir)
			}

			// Find all backup files
			backups, err := findBackups(backupDir)
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}

			if len(backups) == 0 {
				return fmt.Errorf("no backup files found in %s", backupDir)
			}

			// Display backups
			fmt.Println("Available backups:")
			fmt.Println()
			for i, backup := range backups {
				info, _ := os.Stat(backup.Path)
				fmt.Printf("  %d. %s\n", i+1, filepath.Base(backup.Path))
				fmt.Printf("     Created: %s\n", backup.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("     Size: %s\n", formatFileSize(info.Size()))
				fmt.Println()
			}

			// Prompt for selection
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Select backup to restore (enter number): ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			input = strings.TrimSpace(input)
			selection, err := strconv.Atoi(input)
			if err != nil || selection < 1 || selection > len(backups) {
				return fmt.Errorf("invalid selection: %s", input)
			}

			backupPath = backups[selection-1].Path
			fmt.Printf("Selected: %s\n", filepath.Base(backupPath))
		}

		// Verify backup file exists and is readable
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", backupPath)
		}

		// Check if current vault exists and offer to backup it first
		if localStore.Exists() {
			fmt.Print("Current vault exists. Create a backup before restoring? (y/n): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				homeDir, _ := os.UserHomeDir()
				backupDir := filepath.Join(homeDir, ".vaultctl", "backups")
				if err := os.MkdirAll(backupDir, 0700); err != nil {
					return fmt.Errorf("failed to create backup directory: %w", err)
				}
				timestamp := time.Now().Format("2006-01-02T15-04-05Z")
				currentBackupPath := filepath.Join(backupDir, fmt.Sprintf("vault-before-restore-%s.enc", timestamp))

				ev, err := localStore.LoadEncryptedVault()
				if err != nil {
					return fmt.Errorf("failed to load current vault: %w", err)
				}

				data, err := ev.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize vault: %w", err)
				}

				if err := os.WriteFile(currentBackupPath, data, 0600); err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}

				fmt.Printf("Current vault backed up to: %s\n", currentBackupPath)
			}
		}

		// Read the backup file
		backupData, err := os.ReadFile(backupPath)
		if err != nil {
			return fmt.Errorf("failed to read backup file: %w", err)
		}

		// Verify it's valid JSON (basic check)
		// We'll do a more thorough check by trying to parse it
		_, err = storage.EncryptedVaultFromJSON(backupData)
		if err != nil {
			return fmt.Errorf("backup file appears to be invalid or corrupted: %w", err)
		}

		// Ensure vault directory exists
		if err := localStore.EnsureDir(); err != nil {
			return fmt.Errorf("failed to create vault directory: %w", err)
		}

		// Write backup to vault location
		if err := os.WriteFile(cfg.VaultPath, backupData, 0600); err != nil {
			return fmt.Errorf("failed to write restored vault: %w", err)
		}

		fmt.Printf("Vault restored successfully from: %s\n", filepath.Base(backupPath))
		fmt.Println("You can now unlock the vault with: vaultctl unlock")

		return nil
	},
}

// BackupInfo holds information about a backup file
type BackupInfo struct {
	Path      string
	CreatedAt time.Time
}

// findBackups finds all backup files in the backup directory
func findBackups(backupDir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a backup file (ends with .enc or contains "vault-")
		name := entry.Name()
		if !strings.HasSuffix(name, ".enc") && !strings.HasPrefix(name, "vault-") {
			continue
		}

		path := filepath.Join(backupDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      path,
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

