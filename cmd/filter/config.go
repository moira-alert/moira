package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis     cmd.RedisConfig     `yaml:"redis"`
	Logger    cmd.LoggerConfig    `yaml:"log"`
	Filter    filterConfig        `yaml:"filter"`
	Telemetry cmd.TelemetryConfig `yaml:"telemetry"`
}

type filterConfig struct {
	// Metrics listener uri
	Listen string `yaml:"listen"`
	// Retentions config file path.
	// Simply use your original storage-schemas.conf or create new if you're using Moira without existing Graphite installation.
	RetentionConfig string `yaml:"retention_config"`
	// Number of metrics to cache before checking them.
	// Note: As this value increases, Redis CPU usage decreases.
	// Normally, this value must be an order of magnitude less than graphite.prefix.filter.recevied.matching.count | nonNegativeDerivative() | scaleToSeconds(1)
	// For example: with 100 matching metrics, set cache_capacity to 10. With 1000 matching metrics, increase cache_capacity up to 100.
	CacheCapacity int `yaml:"cache_capacity"`
	// Max concurrent metric matchers to run. Equals to the number of processor cores found on Moira host by default or when variable is defined as 0.
	MaxParallelMatches int `yaml:"max_parallel_matches"`
	// Period in which patterns will be reloaded from Redis.
	PatternsUpdatePeriod string `yaml:"patterns_update_period"`
	// DropMetricsTTL this is time window how older metric we can get from now.
	DropMetricsTTL string `yaml:"drop_metrics_ttl"`
	// Flags for compatibility with different graphite behaviours
	Compatibility compatibility `yaml:"graphite_compatibility"`
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Addrs:       "localhost:6379",
			MetricsTTL:  "1h",
			DialTimeout: "500ms",
		},
		Logger: cmd.LoggerConfig{
			LogFile:         "stdout",
			LogLevel:        "info",
			LogPrettyFormat: false,
		},
		Filter: filterConfig{
			Listen:               ":2003",
			RetentionConfig:      "/etc/moira/storage-schemas.conf",
			CacheCapacity:        10, //nolint
			MaxParallelMatches:   0,
			PatternsUpdatePeriod: "1s",
			DropMetricsTTL:       "1h",
			Compatibility: compatibility{
				AllowRegexLooseStartMatch: false,
				AllowRegexMatchEmpty:      true,
			},
		},
		Telemetry: cmd.TelemetryConfig{
			Listen: ":8094",
			Graphite: cmd.GraphiteConfig{
				Enabled:      false,
				RuntimeStats: false,
				URI:          "localhost:2003",
				Prefix:       "DevOps.Moira",
				Interval:     "60s",
			},
			Pprof: cmd.ProfilerConfig{Enabled: false},
		},
	}
}
