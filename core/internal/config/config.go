package config

import (
	"djtracker/internal/utils"
	"fmt"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"
)

type ServerConfig struct {
	BindAddress string `yaml:"bind_address"`
	Port        string
	Format      string
}

type DatabaseConfig struct {
	Path string
}

type TrackerConfig struct {
	History struct {
		Source string
		Path   string
	}
	Source struct {
		Paths []string
	}
}

type ControlConfig struct {
	PinCode string `yaml:"pin_code"`
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Tracker  TrackerConfig
	Control  ControlConfig
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

	r, _ := regexp.Compile("^\\d{6}$")
	if len(c.Control.PinCode) != 6 || !r.MatchString(c.Control.PinCode) {
		return fmt.Errorf("invalid pin code: %s", c.Control.PinCode)
	}

	return nil
}
