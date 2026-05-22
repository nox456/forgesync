package notion

type ProjectProperties struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type StoriesProperties struct {
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
