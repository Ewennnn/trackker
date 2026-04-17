package config

import (
	"djtracker/internal/utils"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server struct {
		BindAddress string `yaml:"bind_address"`
		Port        string
		Format      string
	}
	Database struct {
		Path string
	}
	Tracker struct {
		History struct {
			Source string
			Path   string
		}
		Source struct {
			Paths []string
		}
	}
}

func New() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	var config Config
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Check() error {
	for _, folder := range c.Tracker.Source.Paths {
		if !utils.Exists(folder) {
			return fmt.Errorf("source folder path not found: %s", folder)
		}
	}

	return nil
}
