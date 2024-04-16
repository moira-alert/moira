package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	LogFile         string          `yaml:"log_file"`
	LogLevel        string          `yaml:"log_level"`
	LogPrettyFormat bool            `yaml:"log_pretty_format"`
	Redis           cmd.RedisConfig `yaml:"redis"`
	Cleanup         cleanupConfig   `yaml:"cleanup"`
}

type cleanupConfig struct {
	Whitelist                    []string `yaml:"whitelist"`
	Delete                       bool     `yaml:"delete"`
	AddAnonymousToWhitelist      bool     `json:"add_anonymous_to_whitelist"`
	CleanupMetricsDuration       string   `yaml:"cleanup_metrics_duration"`
	CleanupFutureMetricsDuration string   `yaml:"cleanup_future_metrics_duration"`
}

func getDefault() config {
	return config{
		LogFile:         "stdout",
		LogLevel:        "info",
		LogPrettyFormat: false,
		Redis: cmd.RedisConfig{
			Addrs:       "localhost:6379",
			MetricsTTL:  "1h",
			DialTimeout: "500ms",
		},
		Cleanup: cleanupConfig{
			Whitelist:                    []string{},
			CleanupMetricsDuration:       "-168h",
			CleanupFutureMetricsDuration: "60m",
		},
	}
}
