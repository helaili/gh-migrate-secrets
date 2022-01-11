package cmd

type Secret struct {
	Name                    string `json:"name"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	Visibility              string `json:"visibility"`
	SelectedRepositoriesUrl string `json:"selected_repositories_url"`
	SelectedRepositories    []Repository
	EncryptedValue          string `json:"encrypted_value"`
	KeyId                   string `json:"key_id"`
}

type SecretArrayResponse struct {
	TotalCount int      `json:"total_count"`
	Secrets    []Secret `json:"secrets"`
}

type Repository struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type RepositoryArrayResponse struct {
	TotalCount   int          `json:"total_count"`
	Repositories []Repository `json:"repositories"`
}

type PublicKey struct {
	Id  string `json:"key_id"`
	Key string `json:"key"`
	Raw [32]byte
}

type SecretMigrationOpts struct {
	SourceOrganization       string
	DestinationtOrganization string
}
