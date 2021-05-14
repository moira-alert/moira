package main

import (
	"strings"

	"github.com/moira-alert/moira/database/redis"

	"github.com/xiam/to"
)

type config struct {
	LogFile         string               `mapstructure:"log_file"`
	LogLevel        string               `mapstructure:"log_level"`
	LogPrettyFormat bool                 `mapstructure:"log_pretty_format"`
	Redis           cliRedisConfig       `mapstructure:"redis"`
	Cleanup         cleanupConfig        `mapstructure:"cleanup"`
	CleanupMetrics  cleanupMetricsConfig `mapstructure:"cleanup_metrics"`
}

// Need temporally for viper config reads
type cliRedisConfig struct {
	// Redis Sentinel cluster name
	MasterName string `mapstructure:"master_name"`
	// Redis Sentinel address list, format: {host1_name:port};{ip:port}
	SentinelAddrs string `mapstructure:"sentinel_addrs"`
	// Redis node ip-address or host name
	Host string `mapstructure:"host"`
	// Redis node port
	Port string `mapstructure:"port"`
	// Redis database
	DB              int  `mapstructure:"dbid"`
	ConnectionLimit int  `mapstructure:"connection_limit"`
	AllowSlaveReads bool `mapstructure:"allow_slave_reads"`
	// Moira will delete metrics older than this value from Redis. Large values will lead to various problems everywhere.
	// See https://github.com/moira-alert/moira/pull/519
	MetricsTTL string `mapstructure:"metrics_ttl"`
}

func (config *cliRedisConfig) GetSettings() redis.Config {
	return redis.Config{
		MasterName:        config.MasterName,
		SentinelAddresses: strings.Split(config.SentinelAddrs, ","),
		Host:              config.Host,
		Port:              config.Port,
		DB:                config.DB,
		ConnectionLimit:   config.ConnectionLimit,
		AllowSlaveReads:   config.AllowSlaveReads,
		MetricsTTL:        to.Duration(config.MetricsTTL),
	}
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
		Redis: cliRedisConfig{
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
