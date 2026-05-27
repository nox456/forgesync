package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nox456/forgesync/internal/github"
)

const notionBaseUrl = "https://api.notion.com/v1"
const notionApiVersion = "2026-03-11"

type Client struct {
	Token            string
	ProjectsSourceId string
	StoriesSourceId  string
}

func NewClient(token string, projectsSourceId string, storiesSourceId string) *Client {
	return &Client{
		Token:            token,
		ProjectsSourceId: projectsSourceId,
		StoriesSourceId:  storiesSourceId,
	}
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {

	url := fmt.Sprintf("%s/data_sources/%s/query", notionBaseUrl, c.ProjectsSourceId)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)

	req.Header.Add("Notion-Version", notionApiVersion)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("notion api %d: %s", res.StatusCode, string(b))
	}

	body, _ := io.ReadAll(res.Body)

	var data ProjectsDataSourceResponse

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var projects []Project

	for _, result := range data.Results {
		projects = append(projects, Project{
			PageID: result.ID,
			Name:   result.Properties.Name.Title[0].PlainText,
			Repo:   result.Properties.Repo.RichText[0].PlainText,
		})
	}

	return projects, nil
}

func (c *Client) FindStoryByIssue(ctx context.Context, issue github.Issue) (*Story, error) {

	url := fmt.Sprintf("%s/data_sources/%s/query", notionBaseUrl, c.StoriesSourceId)

	filterPayload := &StoryFilterPayload{
		Filter: PropertyFilter{
			Property: "Issue",
			Number:   NumberFilter{Equals: issue.Number},
		},
	}

	body, err := json.Marshal(filterPayload)

	if err != nil {
		return nil, fmt.Errorf("marshal create page payload: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))

	req.Header.Add("Notion-Version", notionApiVersion)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("notion api %d: %s", res.StatusCode, string(b))
	}

	body, _ = io.ReadAll(res.Body)

	var data StoriesDataSourceResponse

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if len(data.Results) == 0 {
		return nil, nil
	}

	if len(data.Results) > 1 {
		return nil, fmt.Errorf("found more than one story for issue %d", issue.Number)
	}

	var story Story

	for _, result := range data.Results {
		labels := make([]string, len(result.Properties.Labels.MultiSelect))

		for i, label := range result.Properties.Labels.MultiSelect {
			labels[i] = label.Name
		}

		story.PageID = result.ID
		story.Issue = fmt.Sprintf("%d", result.Properties.Issue.Number)
		story.CreatedAt = result.Properties.CreatedTime.Date
		story.Labels = labels
		story.LastWorkedAt = result.Properties.LastWorkedAt.Date.Start
		story.FinishedAt = result.Properties.FinishedDate.Date.Start
		story.Status = result.Properties.Status.Status.Name
		story.Project = result.Properties.Project.Relation[0].ID
		story.Url = result.Properties.URL.URL
		story.Name = result.Properties.Name.Title[0].PlainText
	}

	return &story, nil
}
