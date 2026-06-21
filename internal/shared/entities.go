package shared

import "time"

type Project struct {
	PageID string
	Name   string
	Repo   string
}

type Story struct {
	PageID       string
	Name         string
	Project      string
	Issue        string
	Url          string
	Body         string
	Status       string
	FinishedAt   string
	Labels       []string
	Priority     string
	LastWorkedAt string
	CreatedAt    string
}

type Issue struct {
	Number      int
	Title       string
	URL         string
	Body        string
	State       string
	Labels      []string
	Repo        string
	UpdatedAt   time.Time
	CreatedAt   time.Time
	ClosedAt    *time.Time
	HasLinkedPR bool
}

type UpsertResult struct {
	Created   bool
	Updated   bool
	Unchanged bool
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
