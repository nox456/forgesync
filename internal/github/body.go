package github

import "strings"

func NormalizeGithubBody(input string) string {
	lines := strings.Split(input, "\n")
	var filteredLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}
