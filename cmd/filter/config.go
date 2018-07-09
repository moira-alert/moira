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
	Listen          string `yaml:"listen"`           // Metrics listener uri
	RetentionConfig string `yaml:"retention_config"` // Retentions config file path. Simply use your original storage-schemas.conf or create new if you're using Moira without existing Graphite installation.
	CacheCapacity   int    `yaml:"cache_capacity"`   // Number of metrics to cache before sending them to Redis. Note: As this value increases, Redis CPU usage decreases. Normally, this must be value of the same order of graphite.prefix.filter.recevied.matching.count | nonNegativeDerivative() | scaleToSeconds(1)
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host: "localhost",
			Port: "6379",
			DBID: 0,
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "info",
		},
		Filter: filterConfig{
			Listen:          ":2003",
			RetentionConfig: "/etc/moira/storage-schemas.conf",
			CacheCapacity:   100,
		},
		Graphite: cmd.GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s",
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}
