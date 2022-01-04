package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

func migrateSecrets(opts SecretMigrationOpts) error {
	var args []string
	if opts.SourceOrganization != "" {
		fmt.Printf("Migrating secrets from %s to %s\n", opts.SourceOrganization, opts.DestinationtOrganization)
		args = []string{"api", fmt.Sprintf("/orgs/%s/actions/secrets", opts.SourceOrganization)}
	} else {
		fmt.Printf("Migrating secrets to %s\n", opts.DestinationtOrganization)
		args = []string{"api", "/orgs/{owner}/actions/secrets"}
	}
	stdOut, _, err := gh.Exec(args...)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		var secrets SecretResponse
		err := json.Unmarshal(stdOut.Bytes(), &secrets)
		if err != nil {
			fmt.Println(err)
			return err
		} else {
			fmt.Printf("Found %d secrets\n", secrets.TotalCount)
		}
	}

	return nil
}

func rootCmd() *cobra.Command {
	opts := SecretMigrationOpts{}
	cmd := &cobra.Command{
		Use:   "migrate-secrets",
		Short: "Migrate secrets from an organization or a repo to another",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateSecrets(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.SourceOrganization, "src", "s", "", "Organization with the secrets to migrate")
	cmd.Flags().StringVarP(&opts.DestinationtOrganization, "dest", "d", "", "Organization where the secrets will be migrated")
	cmd.MarkFlagRequired("dest")

	return cmd
}

func main() {
	rc := rootCmd()

	if err := rc.Execute(); err != nil {
		os.Exit(1)
	}
}
