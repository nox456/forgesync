package linear

import (
	"encoding/json"
	"time"
)

type graphqlError struct {
	Message string `json:"message"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors"`
}

type stringComparator struct {
	Eq  string `json:"eq,omitempty"`
	Neq string `json:"neq,omitempty"`
}

type dateComparator struct {
	Gte string `json:"gte"`
}

type workflowStateFilter struct {
	Type stringComparator `json:"type"`
}

type teamFilter struct {
	Key stringComparator `json:"key"`
}

type issueFilter struct {
	UpdatedAt dateComparator      `json:"updatedAt"`
	State     workflowStateFilter `json:"state"`
	Team      *teamFilter         `json:"team,omitempty"`
}

type assignedIssuesVariables struct {
	First  int         `json:"first"`
	After  string      `json:"after,omitempty"`
	Filter issueFilter `json:"filter"`
}

type labelNode struct {
	Name string `json:"name"`
}

type attachmentNode struct {
	SourceType string `json:"sourceType"`
	URL        string `json:"url"`
}

type issueNode struct {
	ID          string     `json:"id"`
	Identifier  string     `json:"identifier"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	State       struct {
		Type string `json:"type"`
	} `json:"state"`
	Team struct {
		Key string `json:"key"`
	} `json:"team"`
	Labels struct {
		Nodes []labelNode `json:"nodes"`
	} `json:"labels"`
	Attachments struct {
		Nodes []attachmentNode `json:"nodes"`
	} `json:"attachments"`
}

type assignedIssuesResponse struct {
	Viewer struct {
		AssignedIssues struct {
			Nodes    []issueNode `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"assignedIssues"`
	} `json:"viewer"`
}
