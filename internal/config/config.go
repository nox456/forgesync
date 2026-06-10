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


// jaaj nose

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
	if viper.GetString("github_token") == "" {
		missingFields = append(missingFields, "github_token")
	}

	if viper.GetString("notion_token") == "" {
		missingFields = append(missingFields, "notion_token")
	}

	if viper.GetString("projects_source_id") == "" {
		missingFields = append(missingFields, "projects_source_id")
	}

	if viper.GetString("stories_source_id") == "" {
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
