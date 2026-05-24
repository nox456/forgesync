package notion

type Project struct {
	PageID string `json:"pageId"`
	Name   string `json:"name"`
	Repo   string `json:"repo"`
}

type Story struct {
	PageID       string   `json:"pageId"`
	Name         string   `json:"name"`
	Project      string   `json:"project"`
	Issue        string   `json:"issue"`
	Url          string   `json:"url"`
	Status       string   `json:"status"`
	FinishedAt   string   `json:"finishedAt"`
	Labels       []string `json:"labels"`
	Priority     string   `json:"priority"`
	LastWorkedAt string   `json:"lastWorkedAt"`
	CreatedAt    string   `json:"createdAt"`
}

type StoryInput struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	Issue   string `json:"issue"`
	Url     string `json:"url"`
	Status  string `json:"status"`
	Labels  []string
}
