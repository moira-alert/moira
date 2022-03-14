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
	Whitelist                         []string `yaml:"whitelist"`
	Delete                            bool     `yaml:"delete"`
	AddAnonymousToWhitelist           bool     `json:"add_anonymous_to_whitelist"`
	CleanupMetricsDuration            string   `yaml:"cleanup_metrics_duration"`
	CleanupMetricsBatchCount          int      `yaml:"cleanup_metrics_batch"`
	CleanupMetricsBatchTimeoutSeconds int      `yaml:"cleanup_metrics_batch_timeout_seconds"`
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
			Whitelist:                         []string{},
			CleanupMetricsDuration:            "-168h",
			CleanupMetricsBatchCount:          100,
			CleanupMetricsBatchTimeoutSeconds: 10,
		},
	}
}
