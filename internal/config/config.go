package config

import (
    "os"
    "gopkg.in/yaml.v3"
)

type RuleConfig struct {
    Enabled  *bool  `yaml:"enabled"`
    Severity string `yaml:"severity"`
}

type Config struct {
    Rules map[string]RuleConfig `yaml:"rules"`
}

func Load(path string) (Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Config{}, err
    }

    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    if err != nil {
        return Config{}, err
    }

    return cfg, nil
}