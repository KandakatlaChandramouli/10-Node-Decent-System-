package types

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Node struct {
		ID   string `yaml:"id"`
		Port int    `yaml:"port"`
	} `yaml:"node"`
	Consensus struct {
		Algorithm  string `yaml:"algorithm"`
		Difficulty int    `yaml:"difficulty"`
	} `yaml:"consensus"`
	Storage struct {
		Backend string `yaml:"backend"`
		Path    string `yaml:"path"`
	} `yaml:"storage"`
	P2P struct {
		Peers []string `yaml:"peers"`
	} `yaml:"p2p"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	return &cfg, nil
}
