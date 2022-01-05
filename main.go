package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

func getOrgPublicKey(opts SecretMigrationOpts) (*PublicKey, error) {
	var publicKeyAPICallArgs []string
	if opts.SourceOrganization != "" {
		publicKeyAPICallArgs = []string{"api", fmt.Sprintf("/orgs/%s/actions/secrets/public-key", opts.SourceOrganization)}
	} else {
		// Use the org name from the current repository
		publicKeyAPICallArgs = []string{"api", "/orgs/{owner}/actions/secrets/public-key"}
	}

	return getPublicKey(publicKeyAPICallArgs)
}

/*
 * The public key is needed to encrypt the secrets.
 */
func getPublicKey(publicKeyAPICallArgs []string) (*PublicKey, error) {
	publicKeyAPICallOutput, _, err := gh.Exec(publicKeyAPICallArgs...)

	if err != nil {
		return nil, err
	} else {
		var publicKey PublicKey
		err := json.Unmarshal(publicKeyAPICallOutput.Bytes(), &publicKey)

		if err != nil {
			return nil, err
		} else {
			decoded, err := base64.StdEncoding.DecodeString(publicKey.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to decode public key: %w", err)
			}
			copy(publicKey.Raw[:], decoded[0:32])
			return &publicKey, nil
		}
	}
}

/*
 * Repo IDs are needed to allow repos to access org level secrets.
 * Potentially, the same repo will get several secret, so we enable caching
 * to avoid retrieiing the same id serveral times.
 */
func getRepoId(destOrg, name string) (int, error) {
	fmt.Printf("Getting id of repository %s/%s.... ", destOrg, name)
	repoAPICallArgs := []string{"api", fmt.Sprintf("repos/%s/%s", destOrg, name), "--cache", "3600s"}
	repoAPICallOutput, _, err := gh.Exec(repoAPICallArgs...)

	if err != nil {
		// Need to handle gracefully when the repo doesn't exist
		if !strings.Contains(err.Error(), "Not Found (HTTP 404)") {
			return -1, err
		} else {
			fmt.Println("Repository not found.")
			return -1, nil
		}
	} else {
		var repo Repository
		err := json.Unmarshal(repoAPICallOutput.Bytes(), &repo)

		if err != nil {
			return -1, err
		} else {
			fmt.Println("Ok.")
			return repo.Id, nil
		}
	}
}

/*
 * A secret applies to srcorg/repo1 and srcorg/repo2. We need to find the ids of
 * destorg/repo1 and destorg/repo2 and create a comma separated list of those ids.
 * It can happen that one or more repos do not exist in destorg.
 */
func getRepoIdList(destOrg, selectedRepositoriesUrl string) (string, error) {
	selectedRepoURLRegex := regexp.MustCompile(`(\/orgs\/.*)`)
	selectedReposAPICallArgs := []string{"api", selectedRepoURLRegex.FindString(selectedRepositoriesUrl)}
	selectedReposAPICallArgsOutput, _, err := gh.Exec(selectedReposAPICallArgs...)

	if err != nil {
		return "", err
	} else {
		var repos RepositoryArrayResponse
		err := json.Unmarshal(selectedReposAPICallArgsOutput.Bytes(), &repos)

		if err != nil {
			return "", err
		} else {
			fmt.Printf("Secret apply to %d repositories\n", repos.TotalCount)
			var repoIdList string

			for _, repo := range repos.Repositories {
				// Check if this repo exist in the target directory and get its id
				repoId, err := getRepoId(destOrg, repo.Name)
				if err != nil {
					return "", err
				} else if repoId != -1 {
					// Add the repo id to the list of repositories to which the secret will be applied
					if len(repoIdList) == 0 {
						repoIdList = fmt.Sprintf("%d", repoId)
					} else {
						repoIdList = fmt.Sprintf("%s,%d", repoIdList, repoId)
					}
				}
			}

			/*
				if len(repoIdList) > 0 {
					fmt.Printf("Repo id list is %s\n", repoIdList)
				}
			*/
			return repoIdList, nil
		}
	}
}

func saveSecret(org string, publicKey PublicKey, secret Secret, repoIdList string) error {
	/*
		  var randomOverride io.Reader
			body := "secret"

			eBody, err := box.SealAnonymous(nil, body, &publicKey.Raw, randomOverride)
			if err != nil {
				return fmt.Errorf("failed to encrypt body: %w", err)
			}

			encoded := base64.StdEncoding.EncodeToString(eBody)
	*/
	return nil
}

func migrateSecrets(opts SecretMigrationOpts) error {
	var orgSecretsAPICallArgs []string

	if opts.SourceOrganization != "" {
		// Use the org name from the command line paramater
		fmt.Printf("Migrating secrets from %s to %s\n", opts.SourceOrganization, opts.DestinationtOrganization)
		orgSecretsAPICallArgs = []string{"api", fmt.Sprintf("/orgs/%s/actions/secrets", opts.SourceOrganization)}
	} else {
		// Use the org name from the current repository
		fmt.Printf("Migrating secrets to %s\n", opts.DestinationtOrganization)
		orgSecretsAPICallArgs = []string{"api", "/orgs/{owner}/actions/secrets"}
	}

	// Get the secrets from the source organization
	orgSecretsAPICallOutput, _, err := gh.Exec(orgSecretsAPICallArgs...)

	if err != nil {
		fmt.Println(err)
		return err
	} else {
		var secrets SecretArrayResponse
		err := json.Unmarshal(orgSecretsAPICallOutput.Bytes(), &secrets)
		if err != nil {
			fmt.Println(err)
			return err
		} else {
			fmt.Printf("Found %d secrets\n", secrets.TotalCount)

			publicKey, err := getOrgPublicKey(opts)
			if err != nil {
				fmt.Println(err)
				return err
			} else {
				fmt.Println("Retrieved public key")
			}

			for _, secret := range secrets.Secrets {
				fmt.Printf("Migrating secret %s\n", secret.Name)

				secret.KeyId = publicKey.Id

				if len(secret.SelectedRepositoriesUrl) > 0 {
					// This secret applies to a list of repositories within the organization
					repoIdList, err := getRepoIdList(opts.DestinationtOrganization, secret.SelectedRepositoriesUrl)
					if err != nil {
						fmt.Println(err)
						return err
					}

					saveSecret(opts.DestinationtOrganization, *publicKey, secret, repoIdList)
				} else {
					saveSecret(opts.DestinationtOrganization, *publicKey, secret, "")
				}
			}
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
