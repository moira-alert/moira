package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	LogFile         string               `mapstructure:"log_file"`
	LogLevel        string               `mapstructure:"log_level"`
	LogPrettyFormat bool                 `mapstructure:"log_pretty_format"`
	Redis           cmd.RedisConfig      `mapstructure:"redis"`
	Cleanup         cleanupConfig        `mapstructure:"cleanup"`
	CleanupMetrics  cleanupMetricsConfig `mapstructure:"cleanup_metrics"`
}

type cleanupConfig struct {
	Whitelist               []string `mapstructure:"whitelist"`
	Delete                  bool     `mapstructure:"delete"`
	AddAnonymousToWhitelist bool     `mapstructure:"add_anonymous_to_whitelist"`
}

type cleanupMetricsConfig struct {
	DryRunMode bool                    `mapstructure:"dryrun_mode"`
	DebugMode  bool                    `mapstructure:"debug_mode"`
	HotParams  cleanupMetricsHotParams `mapstructure:"hot_params"`
}

type cleanupMetricsHotParams struct {
	CleanupDuration            string `mapstructure:"cleanup_duration"`
	CleanupBatchCount          int    `mapstructure:"cleanup_batch"`
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
		CleanupMetrics: cleanupMetricsConfig{
			DryRunMode: true,
			DebugMode:  false,
			HotParams: cleanupMetricsHotParams{
				CleanupDuration:            "-168h",
				CleanupBatchCount:          100,
				CleanupBatchTimeoutSeconds: 10,
			},
		},
	}
}
