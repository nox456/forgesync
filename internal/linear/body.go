package linear

import (
	"regexp"
	"strings"
)

var linearUploadImage = regexp.MustCompile(`!\[[^\]]*\]\(https://uploads\.linear\.app/[^)\s]*(\s+"[^"]*")?\)`)

func normalizeLinearBody(input string) string {
	input = linearUploadImage.ReplaceAllString(input, "")

	lines := strings.Split(input, "\n")
	var filteredLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}
