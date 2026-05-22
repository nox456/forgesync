package notion

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) ListProjects() ([]ProjectProperties, error) {

	url := fmt.Sprintf("%s/data_sources/%s/query", notionBaseUrl, c.ProjectsSourceId)

	req, _ := http.NewRequest("POST", url, nil)

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

	var data DataSourceResponse

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var projects []ProjectProperties

	for _, result := range data.Results {
		projects = append(projects, ProjectProperties{
			Name: result.Properties.Name.Title[0].PlainText,
			Repo: result.Properties.Repo.RichText[0].PlainText,
		})
	}

	return projects, nil
}
