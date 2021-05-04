package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	LogFile         string               `yaml:"log_file"`
	LogLevel        string               `yaml:"log_level"`
	LogPrettyFormat bool                 `yaml:"log_pretty_format"`
	Redis           cmd.RedisConfig      `yaml:"redis"`
	Cleanup         cleanupConfig        `yaml:"cleanup"`
	CleanupMetrics  cleanupMetricsConfig `mapstructure:"cleanup_metrics"`
}

type cleanupConfig struct {
	Whitelist               []string `yaml:"whitelist"`
	Delete                  bool     `yaml:"delete"`
	AddAnonymousToWhitelist bool     `json:"add_anonymous_to_whitelist"`
}

type cleanupMetricsConfig struct {
	DryRunMode bool                    `mapstructure:"dryrun_mode"`
	DebugMode  bool                    `mapstructure:"debug_mode"`
	HotParams  cleanupMetricsHotParams `mapstructure:"hot_params"`
}

type cleanupMetricsHotParams struct {
	CleanupDuration            string `mapstructure:"cleanup_duration"`
	CleanupBatchCount          int    `mapstructure:"cleanup_batch"`
	CleanupKeyScanBatchCount   int    `mapstructure:"cleanup_keyscan_batch"`
	CleanupBatchTimeoutSeconds int    `mapstructure:"cleanup_batch_timeout_seconds"`
}

func getDefault() config {
	return config{
		LogFile:         "stdout",
		LogLevel:        "info",
		LogPrettyFormat: false,
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
