package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	addName       string
	addUsername   string
	addURL        string
	addNotes      string
	addBackupCodes string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new password entry",
	Long:  `Add a new password entry to the vault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(cmd); err != nil {
			return err
		}

		if addName == "" {
			return fmt.Errorf("--name is required")
		}

		// Check if entry already exists
		if unlockedVault.GetEntry(addName) != nil {
			return fmt.Errorf("entry with name '%s' already exists", addName)
		}

		// Prompt for password
		fmt.Print("Enter password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		// Parse backup codes
		var backupCodes []string
		if addBackupCodes != "" {
			// Split by comma, semicolon, or newline
			codes := strings.FieldsFunc(addBackupCodes, func(r rune) bool {
				return r == ',' || r == ';' || r == '\n'
			})
			// Trim whitespace from each code
			for _, code := range codes {
				trimmed := strings.TrimSpace(code)
				if trimmed != "" {
					backupCodes = append(backupCodes, trimmed)
				}
			}
		} else {
			// Prompt interactively for backup codes (optional)
			fmt.Print("Enter backup codes? (y/n, or press Enter to skip): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			
			if response == "y" || response == "yes" {
				fmt.Println("Enter backup codes (one per line, empty line to finish):")
				for {
					fmt.Print("  Code: ")
					code, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					code = strings.TrimSpace(code)
					if code == "" {
						break
					}
					backupCodes = append(backupCodes, code)
				}
			}
		}

		// Add entry
		unlockedVault.AddEntry(addName, addUsername, string(password), addURL, addNotes, backupCodes)

		// Save vault
		sync := !cmd.Flags().Changed("no-sync")
		if err := saveVault(cmd, sync); err != nil {
			return fmt.Errorf("failed to save vault: %w", err)
		}

		fmt.Printf("Entry '%s' added successfully\n", addName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&addName, "name", "", "Entry name (required)")
	addCmd.Flags().StringVar(&addUsername, "username", "", "Username")
	addCmd.Flags().StringVar(&addURL, "url", "", "URL")
	addCmd.Flags().StringVar(&addNotes, "notes", "", "Notes")
	addCmd.Flags().StringVar(&addBackupCodes, "backup-codes", "", "2FA backup codes (comma or semicolon separated, or leave empty for interactive input)")
	addCmd.Flags().Bool("no-sync", false, "Don't sync to DynamoDB")
}

