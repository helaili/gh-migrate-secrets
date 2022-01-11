package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

var sourceOrganization string
var outputFile string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the definitions of secrets",
	Run: func(cmd *cobra.Command, args []string) {
		if sourceOrganization == "" {
			repo, _ := gh.CurrentRepository()
			sourceOrganization = repo.Owner()
		}

		exportSecrets(sourceOrganization)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&sourceOrganization, "sourceOrg", "s", "", "Organization with the secrets to migrate")
	exportCmd.Flags().StringVarP(&outputFile, "outputFile", "o", "secrets.csv", "File to write the output to")
}

func dumpSecrets(secrets *[]Secret) error {
	header := []string{"organization", "name", "visibility", "selected repositories", "value"}
	csvFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed creating file: %w", err)
	}

	csvwriter := csv.NewWriter(csvFile)

	csvwriter.Write(header)
	for _, secret := range *secrets {
		_ = csvwriter.Write(dumpSecret(secret))
	}
	csvwriter.Flush()
	csvFile.Close()
	return nil
}

func dumpSecret(secret Secret) []string {
	var selectedRepoList string

	if len(secret.SelectedRepositories) > 0 {
		for _, repo := range secret.SelectedRepositories {
			selectedRepoList = fmt.Sprintf("%s %s", selectedRepoList, repo.Name)
		}
	}

	return []string{sourceOrganization, secret.Name, secret.Visibility, selectedRepoList}
}

func retrieveRepoList(secret *Secret) error {
	selectedRepoURLRegex := regexp.MustCompile(`(\/orgs\/.*)`)
	// selectedReposAPICallArgs := []string{"api", selectedRepoURLRegex.FindString(secret.SelectedRepositoriesUrl)}
	selectedReposAPICallArgs := []string{"api", selectedRepoURLRegex.FindString(strings.Replace(secret.SelectedRepositoriesUrl, "dependabot", "actions", 1))}
	selectedReposAPICallArgsOutput, _, err := gh.Exec(selectedReposAPICallArgs...)

	if err != nil {
		return fmt.Errorf("failed getting the list of repositories for the secret %s - %w", secret.Name, err)
	}

	var repos RepositoryArrayResponse
	unmarshallErr := json.Unmarshal(selectedReposAPICallArgsOutput.Bytes(), &repos)

	if unmarshallErr != nil {
		return fmt.Errorf("failed unmarshalling the list of repositories for the secret %s - %w", secret.Name, unmarshallErr)
	}
	secret.SelectedRepositories = repos.Repositories

	return nil
}

func exportSecrets(sourceOrganization string) error {
	var orgSecretsAPICallArgs []string

	fmt.Printf("Exporting secrets from %s\n", sourceOrganization)
	orgSecretsAPICallArgs = []string{"api", fmt.Sprintf("/orgs/%s/actions/secrets", sourceOrganization)}

	// Get the secrets from the source organization
	orgSecretsAPICallOutput, _, err := gh.Exec(orgSecretsAPICallArgs...)

	if err != nil {
		fmt.Println(err)
		return err
	}
	var secrets SecretArrayResponse
	marshalErr := json.Unmarshal(orgSecretsAPICallOutput.Bytes(), &secrets)

	if marshalErr != nil {
		fmt.Println(marshalErr)
		return marshalErr
	}
	fmt.Printf("Found %d secrets\n", secrets.TotalCount)

	for idx, _ := range secrets.Secrets {
		if len(secrets.Secrets[idx].SelectedRepositoriesUrl) > 0 {
			// This secret applies to a list of repositories within the organization
			err := retrieveRepoList(&secrets.Secrets[idx])

			if err != nil {
				fmt.Println(err)
				return err
			}
		}
	}

	return dumpSecrets(&secrets.Secrets)
}
