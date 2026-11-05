package config

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidate(t *testing.T) {
	// complete builds a config with every Notion key set, so each case only
	// varies the tokens or blanks the key it tests.
	complete := func(githubToken, linearToken string) Config {
		return Config{
			GitHubToken:      githubToken,
			LinearToken:      linearToken,
			NotionToken:      "notion-token",
			ProjectsSourceId: "projects-id",
			StoriesSourceId:  "stories-id",
		}
	}

	cases := []struct {
		name    string
		config  Config
		wantErr []string
	}{
		// happy path: one provider token is enough
		{name: "only a GitHub token is valid", config: complete("gh-token", "")},
		{name: "only a Linear token is valid", config: complete("", "lin-token")},
		{name: "both tokens are valid", config: complete("gh-token", "lin-token")},
		// edge cases
		{
			name: "every missing Notion key is named, not just the first",
			config: Config{
				GitHubToken: "gh-token",
			},
			wantErr: []string{"notion_token", "projects_source_id", "stories_source_id"},
		},
		// error path
		{
			name:    "neither token names both keys",
			config:  complete("", ""),
			wantErr: []string{"github_token", "linear_token"},
		},
		{
			name: "a missing notion_token is still required",
			config: func() Config {
				c := complete("gh-token", "")
				c.NotionToken = ""
				return c
			}(),
			wantErr: []string{"notion_token"},
		},
		{
			name: "a missing projects_source_id is still required",
			config: func() Config {
				c := complete("", "lin-token")
				c.ProjectsSourceId = ""
				return c
			}(),
			wantErr: []string{"projects_source_id"},
		},
		{
			name: "a missing stories_source_id is still required",
			config: func() Config {
				c := complete("gh-token", "lin-token")
				c.StoriesSourceId = ""
				return c
			}(),
			wantErr: []string{"stories_source_id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.config)

			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Errorf("validate() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validate() = nil, want an error naming %v", tc.wantErr)
			}
			for _, key := range tc.wantErr {
				if !strings.Contains(err.Error(), key) {
					t.Errorf("validate() error = %q, want it to name %q", err, key)
				}
			}
		})
	}
}

func TestShouldLoadConfig(t *testing.T) {
	// error path: none — ShouldLoadConfig is total and has no failure mode.
	cases := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		// happy path
		{
			name:        "command in the internal group skips config loading",
			annotations: map[string]string{"group": "internal"},
			want:        false,
		},
		{
			name:        "command with no annotations loads config",
			annotations: nil,
			want:        true,
		},
		// edge cases
		{
			name:        "command with an empty annotations map loads config",
			annotations: map[string]string{},
			want:        true,
		},
		{
			name:        "command in another group loads config",
			annotations: map[string]string{"group": "sync"},
			want:        true,
		},
		{
			name:        "group match is case-sensitive",
			annotations: map[string]string{"group": "Internal"},
			want:        true,
		},
		{
			name:        "empty group value loads config",
			annotations: map[string]string{"group": ""},
			want:        true,
		},
		{
			name:        "internal value under a different key loads config",
			annotations: map[string]string{"category": "internal"},
			want:        true,
		},
		{
			name:        "internal group alongside other annotations skips config loading",
			annotations: map[string]string{"group": "internal", "other": "value"},
			want:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test", Annotations: tc.annotations}

			got := ShouldLoadConfig(cmd)
			if got != tc.want {
				t.Errorf("ShouldLoadConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}
