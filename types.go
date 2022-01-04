package main

type Secret struct {
	Name                    string `json:"name"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	Visibility              string `json:"visibility"`
	SelectedRepositoriesUrl string `json:"selected_repositories_url"`
}

type SecretResponse struct {
	TotalCount int      `json:"total_count"`
	Secrets    []Secret `json:"secrets"`
}

type SecretMigrationOpts struct {
	SourceOrganization       string
	DestinationtOrganization string
}
