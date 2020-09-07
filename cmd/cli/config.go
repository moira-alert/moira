package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	LogFile  string          `yaml:"log_file"`
	LogLevel string          `yaml:"log_level"`
	Redis    cmd.RedisConfig `yaml:"redis"`
	Cleanup  cleanupConfig   `yaml:"cleanup"`
}

type cleanupConfig struct {
	Whitelist               []string `yaml:"whitelist"`
	Delete                  bool     `yaml:"delete"`
	AddAnonymousToWhitelist bool     `json:"add_anonymous_to_whitelist"`
}

func getDefault() config {
	return config{
		LogFile:  "stdout",
		LogLevel: "info",
		Redis: cmd.RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			ConnectionLimit: 512, //nolint
			MetricsTTL:      "1h",
		},
		Cleanup: cleanupConfig{
			Whitelist: []string{},
		},
	}
}
