package github

import "testing"

func TestNormalizeGithubBody(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// happy path
		{
			name:  "blank lines between paragraphs are removed",
			input: "First paragraph\n\nSecond paragraph\n\n\nThird",
			want:  "First paragraph\nSecond paragraph\nThird",
		},
		{
			name:  "a body without blank lines is unchanged",
			input: "- [ ] task one\n- [ ] task two",
			want:  "- [ ] task one\n- [ ] task two",
		},
		// edge cases
		{
			name:  "whitespace-only lines count as blank",
			input: "Line one\n   \n\t\nLine two",
			want:  "Line one\nLine two",
		},
		{
			name:  "an empty body stays empty",
			input: "",
			want:  "",
		},
		{
			name:  "a body of only blank lines becomes empty",
			input: "\n\n  \n",
			want:  "",
		},
		{
			name:  "indentation inside kept lines is preserved",
			input: "```go\n    func main() {}\n```",
			want:  "```go\n    func main() {}\n```",
		},
		// error path: none — normalizeGithubBody is total and cannot fail.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeGithubBody(tc.input); got != tc.want {
				t.Errorf("normalizeGithubBody(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
