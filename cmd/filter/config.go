package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Filter   filterConfig       `yaml:"filter"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
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
	// Allow to receive compressed(snappy and gzip are supported) metrics
	Compression string `yaml:"compression"`
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			ConnectionLimit: 512,
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "info",
		},
		Filter: filterConfig{
			Listen:             ":2003",
			RetentionConfig:    "/etc/moira/storage-schemas.conf",
			CacheCapacity:      10,
			MaxParallelMatches: 0,
			Compression:        "",
		},
		Graphite: cmd.GraphiteConfig{
			RuntimeStats: false,
			URI:          "localhost:2003",
			Prefix:       "DevOps.Moira",
			Interval:     "60s",
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}
