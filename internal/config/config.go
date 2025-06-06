package config

import (
	"os"
)

type Config struct {
	ProjectID string
	Location  string
	Model     string
}

func Load() (*Config, error) {
	projectID := os.Getenv("VERTEX_AI_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	location := os.Getenv("VERTEX_AI_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	model := os.Getenv("GEMINIELF_DEFAULT_MODEL")
	if model == "" {
		model = "gemini-2.5-flash-preview-05-20"
	}

	return &Config{
		ProjectID: projectID,
		Location:  location,
		Model:     model,
	}, nil
}