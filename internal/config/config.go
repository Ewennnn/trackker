package config

import (
	"github.com/goccy/go-yaml"
	"os"
)

type Config struct {
	Server struct {
		BindAddress string `yaml:"bind_address"`
		Port        string
	}
	Database struct {
		Path string
	}
	Tracker struct {
		Path string
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
