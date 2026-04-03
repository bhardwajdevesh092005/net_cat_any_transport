package cli

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Options    GlobalOptions        `yaml:"options"`
	Transports map[string]Transport `yaml:"transports"`
}

type GlobalOptions struct {
	Timeout int  `yaml:"timeout"`  // in seconds
	Verbose bool `yaml:"verbose"`
}

type Transport struct {
	Type     string `yaml:"type"`     // tcp, udp, ws...
	Listen   string `yaml:"listen"`   // Address to listen on
	Connect  string `yaml:"connect"`  // Address to connect to
	Exec     string `yaml:"exec"`     // Command to execute
	ZeroIO   bool   `yaml:"zero_io"`  // Scan mode
	TLS      bool   `yaml:"tls"`      // Encrypt socket via TLS overlaid
	Password string `yaml:"password"` // Expected PSK password for session auth
}

// LoadConfig reads the YAML configuration file from path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
