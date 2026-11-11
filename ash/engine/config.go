package engine

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	Cmd    string            `yaml:"cmd"`
	Args   []string          `yaml:"args"`
	Flags  map[string]string `yaml:"flags"`
	Prompt string            `yaml:"prompt"`
}

type AgentsConfig struct {
	Agents map[string]AgentConfig `yaml:"agents"`
}

//go:embed agents.yaml
var embeddedData []byte

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ash", "agents.yaml")
}

func LoadConfig() (*AgentsConfig, error) {
	paths := []string{}

	if env := os.Getenv("ASH_AGENTS_CONFIG"); env != "" {
		paths = append(paths, env)
	}

	if p := defaultConfigPath(); p != "" {
		paths = append(paths, p)
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		paths = append(paths, filepath.Join(cwd, ".ash-agents.yaml"))
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			cfg := &AgentsConfig{}
			if yamlErr := yaml.Unmarshal(data, cfg); yamlErr != nil {
				return nil, fmt.Errorf("parsing agents config %s: %w", path, yamlErr)
			}
			return cfg, nil
		}
	}

	cfg := &AgentsConfig{}
	if yamlErr := yaml.Unmarshal(embeddedData, cfg); yamlErr != nil {
		return nil, fmt.Errorf("parsing embedded agents config: %w", yamlErr)
	}
	return cfg, nil
}
