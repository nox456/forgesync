package linear

import "testing"

func TestNormalizeLinearBody(t *testing.T) {
	const upload = "https://uploads.linear.app/8f2e/4b1a/9c3d"

	cases := []struct {
		name  string
		input string
		want  string
	}{
		// happy path
		{
			name:  "a Linear image on its own line is removed",
			input: "Steps:\n![screenshot.png](" + upload + ")\nThen click save",
			want:  "Steps:\nThen click save",
		},
		{
			name:  "text around an inline Linear image is kept",
			input: "See ![diagram](" + upload + ") for details",
			want:  "See  for details",
		},
		{
			name:  "an image hosted elsewhere is kept",
			input: "![logo](https://example.com/logo.png)",
			want:  "![logo](https://example.com/logo.png)",
		},
		// edge cases
		{
			name:  "a Linear image with a title is removed",
			input: "Before\n![shot](" + upload + ` "Screenshot")` + "\nAfter",
			want:  "Before\nAfter",
		},
		{
			name:  "only the Linear images are removed when hosts are mixed",
			input: "![a](" + upload + ")\n![b](https://i.imgur.com/b.png)\n![c](" + upload + "/2)",
			want:  "![b](https://i.imgur.com/b.png)",
		},
		{
			name:  "a plain link to a Linear upload is kept",
			input: "[report.pdf](" + upload + ")",
			want:  "[report.pdf](" + upload + ")",
		},
		{
			name:  "a host that only starts with uploads.linear.app is kept",
			input: "![x](https://uploads.linear.app.example.com/y.png)",
			want:  "![x](https://uploads.linear.app.example.com/y.png)",
		},
		{
			name:  "blank and whitespace-only lines are dropped",
			input: "First\n\n   \n\t\nSecond",
			want:  "First\nSecond",
		},
		{
			name:  "an empty description stays empty",
			input: "",
			want:  "",
		},
		// error path: none — normalizeLinearBody is total and cannot fail.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeLinearBody(tc.input); got != tc.want {
				t.Errorf("normalizeLinearBody(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
