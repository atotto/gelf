package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectID string `yaml:"project_id"`
	Location  string `yaml:"location"`
	Model     string `yaml:"model"`
}

type FileConfig struct {
	VertexAI struct {
		ProjectID string `yaml:"project_id"`
		Location  string `yaml:"location"`
	} `yaml:"vertex_ai"`
	Gelf struct {
		DefaultModel string `yaml:"default_model"`
	} `yaml:"gelf"`
}

func Load() (*Config, error) {
	// Load from file first (lowest priority)
	fileConfig, err := loadFromFile()
	if err != nil {
		// File not found or invalid is not an error - use defaults
		fileConfig = &FileConfig{}
	}

	// Environment variables override file config
	projectID := os.Getenv("VERTEXAI_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		projectID = fileConfig.VertexAI.ProjectID
	}

	location := os.Getenv("VERTEXAI_LOCATION")
	if location == "" {
		location = fileConfig.VertexAI.Location
	}
	if location == "" {
		location = "us-central1"
	}

	model := fileConfig.Gelf.DefaultModel
	if model == "" {
		model = "gemini-2.5-flash-preview-05-20"
	}

	return &Config{
		ProjectID: projectID,
		Location:  location,
		Model:     model,
	}, nil
}

func loadFromFile() (*FileConfig, error) {
	// Try to find gelf.yml in current directory, XDG config, or home directory
	configPaths := []string{
		"gelf.yml",
		"gelf.yaml",
	}
	
	// Add XDG config directory paths
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		configPaths = append(configPaths,
			filepath.Join(xdgConfigHome, "gelf", "gelf.yml"),
			filepath.Join(xdgConfigHome, "gelf", "gelf.yaml"),
		)
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		// Fallback to ~/.config if XDG_CONFIG_HOME is not set
		configPaths = append(configPaths,
			filepath.Join(homeDir, ".config", "gelf", "gelf.yml"),
			filepath.Join(homeDir, ".config", "gelf", "gelf.yaml"),
		)
	}
	
	// Add home directory paths
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPaths = append(configPaths,
			filepath.Join(homeDir, ".gelf.yml"),
			filepath.Join(homeDir, ".gelf.yaml"),
		)
	}

	var config FileConfig
	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Try next path
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
		return &config, nil
	}

	return nil, os.ErrNotExist
}