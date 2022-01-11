package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var destinationOrg string

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import secrets",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Import")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&destinationOrg, "org", "o", "", "Organization where the secrets will be migrated")
	importCmd.MarkFlagRequired("org")
}
