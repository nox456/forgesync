package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	GitHubToken      string
	NotionToken      string
	ProjectsSourceId string
	StoriesSourceId  string
}

func Load() (*Config, error) {
	// Load env vars
	viper.SetEnvPrefix("FORGESYNC")
	viper.AutomaticEnv()

	// Load config file
	viper.SetConfigName("config")
	viper.AddConfigPath("$XDG_CONFIG_HOME/forgesync")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var missingFields []string

	// Validate required fields
	if viper.Get("github_token") == nil {
		missingFields = append(missingFields, "github_token")
	}

	if viper.Get("notion_token") == nil {
		missingFields = append(missingFields, "notion_token")
	}

	if viper.Get("projects_source_id") == nil {
		missingFields = append(missingFields, "projects_source_id")
	}

	if viper.Get("stories_source_id") == nil {
		missingFields = append(missingFields, "stories_source_id")
	}

	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing required fields: %s", missingFields)
	}

	return &Config{
		GitHubToken:      viper.GetString("github_token"),
		NotionToken:      viper.GetString("notion_token"),
		ProjectsSourceId: viper.GetString("projects_source_id"),
		StoriesSourceId:  viper.GetString("stories_source_id"),
	}, nil
}
