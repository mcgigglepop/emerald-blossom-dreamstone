package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all password entries",
	Long:  `List all password entries in the vault (without showing passwords).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(cmd); err != nil {
			return err
		}

		entries := unlockedVault.ListEntries()
		if len(entries) == 0 {
			fmt.Println("No entries found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tUSERNAME\tURL\tUPDATED")
		for _, entry := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				entry.Name,
				entry.Username,
				entry.URL,
				entry.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

