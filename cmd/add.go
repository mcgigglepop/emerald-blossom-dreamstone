package cmd

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	addName     string
	addUsername string
	addURL      string
	addNotes    string
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

		// Add entry
		unlockedVault.AddEntry(addName, addUsername, string(password), addURL, addNotes)

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
	addCmd.Flags().Bool("no-sync", false, "Don't sync to DynamoDB")
}

