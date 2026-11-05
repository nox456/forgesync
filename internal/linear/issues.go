package linear

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nox456/forgesync/internal/shared"
)

const assignedIssuesQuery = `query AssignedIssues($first: Int!, $after: String, $filter: IssueFilter) {
	viewer {
		assignedIssues(first: $first, after: $after, filter: $filter, orderBy: updatedAt) {
			nodes {
				id
				identifier
				title
				url
				description
				createdAt
				updatedAt
				completedAt
				state { type }
				team { key }
				labels { nodes { name } }
				attachments { nodes { sourceType url } }
			}
			pageInfo { hasNextPage endCursor }
		}
	}
}`

func (c *Client) FetchAssignedIssues(ctx context.Context, teamKey string) ([]shared.Issue, error) {
	var issues []shared.Issue

	variables := assignedIssuesVariables{
		First: linearPageSize,
		Filter: issueFilter{
			UpdatedAt: dateComparator{Gte: c.Now().AddDate(0, 0, -7).Format(time.RFC3339)},
			State:     workflowStateFilter{Type: stringComparator{Neq: "triage"}},
		},
	}

	if teamKey != "" {
		variables.Filter.Team = &teamFilter{Key: stringComparator{EqIgnoreCase: teamKey}}
	}

	for {
		var data assignedIssuesResponse

		if err := c.request(ctx, assignedIssuesQuery, variables, &data); err != nil {
			return nil, err
		}

		connection := data.Viewer.AssignedIssues

		for _, node := range connection.Nodes {
			issue := toIssue(node)
			slog.Debug(fmt.Sprintf("[LINEAR]: Issue found - Identifier: %s Title: %s Scope: %s", issue.Identifier, issue.Title, issue.Scope))
			issues = append(issues, issue)
		}

		if !connection.PageInfo.HasNextPage || connection.PageInfo.EndCursor == "" {
			break
		}

		variables.After = connection.PageInfo.EndCursor
	}

	return issues, nil
}

func toIssue(node issueNode) shared.Issue {
	labels := make([]string, len(node.Labels.Nodes))
	for i, label := range node.Labels.Nodes {
		labels[i] = label.Name
	}

	return shared.Issue{
		Provider:   shared.ProviderLinear,
		Identifier: node.Identifier,
		Title:      node.Title,
		URL:        node.URL,
		Body:       normalizeLinearBody(node.Description),
		State:      node.State.Type,
		Labels:     labels,
		Scope:      node.Team.Key,
		UpdatedAt:  node.UpdatedAt,
		CreatedAt:  node.CreatedAt,
		ClosedAt:   node.CompletedAt,
	}
}
