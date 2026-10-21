package shared

import "time"

type Provider string

const (
	ProviderGithub Provider = "github"
	ProviderLinear Provider = "linear"
)

type Project struct {
	PageID   string
	Name     string
	Repo     string
	TeamKey  string
	Provider Provider
}

type ProjectFilter struct {
	Provider Provider
	Repo     string
	TeamKey  string
}

func (p Project) GetScope() string {
	if p.Provider == ProviderGithub {
		return p.Repo
	}

	return p.TeamKey
}

type Story struct {
	PageID       string
	Name         string
	Project      string
	Issue        string
	Url          string
	Status       string
	FinishedAt   string
	Labels       []string
	LastWorkedAt string
	CreatedAt    string
}

type Issue struct {
	Title       string
	Identifier  string
	URL         string
	Body        string
	State       string
	Provider    Provider
	Labels      []string
	Scope       string
	UpdatedAt   time.Time
	CreatedAt   time.Time
	ClosedAt    *time.Time
	HasLinkedPR bool
}

type UpsertResult struct {
	Created bool
	Updated bool
}

type StoryInput struct {
	Name         string `json:"name"`
	Project      string `json:"project"`
	Issue        string `json:"issue"`
	Url          string `json:"url"`
	Body         string `json:"body"`
	Status       string `json:"status"`
	Labels       []string
	LastWorkedAt string
	FinishedDate string
}
