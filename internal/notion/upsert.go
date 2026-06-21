package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/utils"
)

func (c *Client) UpsertStory(ctx context.Context, storyInput shared.StoryInput, issue shared.Issue, isDryRun bool, existingStory *shared.Story) (*shared.UpsertResult, error) {
	labels := make([]NamedOption, len(storyInput.Labels))
	for i, l := range storyInput.Labels {
		labels[i] = NamedOption{Name: l}
	}

	baseProps := StoryProperties{
		Name: &TitleInputProperty{
			Title: []RichTextInput{
				{Text: TextContent{Content: storyInput.Name}},
			},
		},
		Status: &StatusProperty{Status: NamedOption{Name: storyInput.Status}},
		Labels: &MultiSelectProperty{MultiSelect: labels},
		LastWorkedAt: &DateProperty{
			Date: DateValue{
				Start: storyInput.LastWorkedAt,
			},
		},
	}

	if storyInput.FinishedDate != "" {
		baseProps.FinishedDate = &DateProperty{
			Date: DateValue{
				Start: storyInput.FinishedDate,
			},
		}
	}

	if existingStory == nil {

		if isDryRun {
			return &shared.UpsertResult{
				Created: true,
			}, nil
		}

		url := fmt.Sprintf("%s/pages", notionBaseUrl)

		createProps := baseProps
		createProps.Project = &RelationProperty{
			Relation: []RelationRef{{ID: storyInput.Project}},
		}
		createProps.Issue = &NumberProperty{Number: issue.Number}
		createProps.URL = &URLProperty{URL: storyInput.Url}

		createPayload := &StoryCreatePayload{
			Parent:     PageParent{DataSourceID: c.StoriesSourceId},
			Properties: createProps,
			Markdown:   storyInput.Body,
		}

		body, err := json.Marshal(createPayload)
		if err != nil {
			return nil, err
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

		slog.Debug("notion story created", "issue", createProps.Issue)

		return &shared.UpsertResult{
			Created: true,
		}, nil
	} else {

		if utils.IsSynced(issue, existingStory) {
			return &shared.UpsertResult{
				Unchanged: true,
			}, nil
		}

		if isDryRun {
			return &shared.UpsertResult{
				Updated: true,
			}, nil
		}

		url := fmt.Sprintf("%s/pages/%s", notionBaseUrl, existingStory.PageID)

		updatePayload := &StoryUpdatePayload{
			Properties: baseProps,
		}

		body, err := json.Marshal(updatePayload)
		if err != nil {
			return nil, err
		}

		req, _ := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))

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

		if storyInput.Body != "" {
			slog.Debug("[SYNC]: Updating story body")
			url = fmt.Sprintf("%s/pages/%s/markdown", notionBaseUrl, existingStory.PageID)

			updateMarkdownPayload := &StoryMarkdownUpdatePayload{
				Type:           "replace_content",
				ReplaceContent: ReplaceContent{NewStr: storyInput.Body},
			}
			body, err = json.Marshal(updateMarkdownPayload)
			if err != nil {
				return nil, err
			}

			req, _ = http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))

			req.Header.Add("Notion-Version", notionApiVersion)
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
			req.Header.Add("Content-Type", "application/json")

			res, err = http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()

			if res.StatusCode >= 400 {
				b, _ := io.ReadAll(res.Body)
				return nil, fmt.Errorf("notion api %d: %s", res.StatusCode, string(b))
			}
		}

		return &shared.UpsertResult{
			Updated: true,
		}, nil
	}
}
