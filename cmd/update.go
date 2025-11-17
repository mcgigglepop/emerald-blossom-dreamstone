package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/crypto"
	"golang.org/x/term"
)

var (
	updateName       string
	updateUsername  string
	updatePassword  string
	updateURL       string
	updateNotes     string
	updateBackupCodes string
)

var updateCmd = &cobra.Command{
	Use:   "update <name_or_id>",
	Short: "Update an existing password entry",
	Long:  `Update fields of an existing password entry. Only provided fields will be updated.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(cmd); err != nil {
			return err
		}

		entry := unlockedVault.GetEntry(args[0])
		if entry == nil {
			return fmt.Errorf("entry not found: %s", args[0])
		}

		// Parse backup codes if provided
		var backupCodes []string
		if updateBackupCodes != "" {
			// Split by comma, semicolon, or newline
			codes := strings.FieldsFunc(updateBackupCodes, func(r rune) bool {
				return r == ',' || r == ';' || r == '\n'
			})
			// Trim whitespace from each code
			for _, code := range codes {
				trimmed := strings.TrimSpace(code)
				if trimmed != "" {
					backupCodes = append(backupCodes, trimmed)
				}
			}
		} else if cmd.Flags().Changed("backup-codes") {
			// Flag was explicitly set to empty, clear backup codes
			backupCodes = []string{}
		}

		// Handle password update
		var password []byte
		if cmd.Flags().Changed("password") {
			if updatePassword == "" {
				// Password flag was set but empty, prompt for new password
				fmt.Print("Enter new password: ")
				pwd, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
				fmt.Println()
				password = pwd
			} else {
				// Password provided via flag (less secure, but supported)
				password = []byte(updatePassword)
			}
		}

		// Update entry
		var codesToUpdate []string
		if backupCodes != nil {
			codesToUpdate = backupCodes
		}
		
		if !unlockedVault.UpdateEntry(args[0], updateName, updateUsername, password, updateURL, updateNotes, codesToUpdate) {
			// Zeroize password if update failed
			if password != nil {
				crypto.Zeroize(password)
			}
			return fmt.Errorf("failed to update entry")
		}
		
		// Zeroize password from memory after use
		if password != nil {
			crypto.Zeroize(password)
		}

		// Save vault
		sync := !cmd.Flags().Changed("no-sync")
		if err := saveVault(cmd, sync); err != nil {
			return fmt.Errorf("failed to save vault: %w", err)
		}

		fmt.Printf("Entry '%s' updated successfully\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVar(&updateName, "name", "", "Update entry name")
	updateCmd.Flags().StringVar(&updateUsername, "username", "", "Update username")
	updateCmd.Flags().StringVar(&updatePassword, "password", "", "Update password (leave empty to prompt securely)")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "Update URL")
	updateCmd.Flags().StringVar(&updateNotes, "notes", "", "Update notes")
	updateCmd.Flags().StringVar(&updateBackupCodes, "backup-codes", "", "Update backup codes (comma or semicolon separated, or empty string to clear)")
	updateCmd.Flags().Bool("no-sync", false, "Don't sync to DynamoDB")
}

