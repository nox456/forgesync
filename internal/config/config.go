package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	GitHubToken      string
	LinearToken      string
	NotionToken      string
	ProjectsSourceId string
	StoriesSourceId  string
}

func Load() (*Config, error) {
	// Load env vars
	viper.SetEnvPrefix("FORGESYNC")
	viper.AutomaticEnv()

	// Load config file
	viper.SetConfigName("config_test")
	viper.AddConfigPath("$XDG_CONFIG_HOME/forgesync")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	config := Config{
		GitHubToken:      viper.GetString("github_token"),
		LinearToken:      viper.GetString("linear_token"),
		NotionToken:      viper.GetString("notion_token"),
		ProjectsSourceId: viper.GetString("projects_source_id"),
		StoriesSourceId:  viper.GetString("stories_source_id"),
	}

	err = validate(config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func ShouldLoadConfig(cmd *cobra.Command) bool {
	group := cmd.Annotations["group"]

	if group == "internal" {
		return false
	}

	return true
}

func validate(config Config) error {
	var missingFields []string

	// Validate required fields
	if config.GitHubToken == "" && config.LinearToken == "" {
		return fmt.Errorf("at least one of github_token, linear_token is required")
	}

	if config.NotionToken == "" {
		missingFields = append(missingFields, "notion_token")
	}

	if config.ProjectsSourceId == "" {
		missingFields = append(missingFields, "projects_source_id")
	}

	if config.StoriesSourceId == "" {
		missingFields = append(missingFields, "stories_source_id")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %s", missingFields)
	}

	return nil
}
