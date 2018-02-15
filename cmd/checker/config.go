package main

import (
	"runtime"

	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
	"github.com/gosexy/to"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Checker  checkerConfig      `yaml:"checker"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
}

type checkerConfig struct {
	NoDataCheckInterval  string `yaml:"nodata_check_interval"`
	CheckInterval        string `yaml:"check_interval"`
	MetricsTTL           string `yaml:"metrics_ttl"`
	StopCheckingInterval string `yaml:"stop_checking_interval"`
	MaxParallelChecks    int    `yaml:"max_parallel_checks"`
}

func (config *checkerConfig) getSettings() *checker.Config {
	if config.MaxParallelChecks == 0 {
		config.MaxParallelChecks = runtime.NumCPU()
	}
	return &checker.Config{
		MetricsTTLSeconds:           int64(to.Duration(config.MetricsTTL).Seconds()),
		CheckInterval:               to.Duration(config.CheckInterval),
		NoDataCheckInterval:         to.Duration(config.NoDataCheckInterval),
		StopCheckingIntervalSeconds: int64(to.Duration(config.StopCheckingInterval).Seconds()),
		MaxParallelChecks:           config.MaxParallelChecks,
	}
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "debug",
		},
		Checker: checkerConfig{
			NoDataCheckInterval:  "60s",
			CheckInterval:        "5s",
			MetricsTTL:           "1h",
			StopCheckingInterval: "30s",
			MaxParallelChecks:    0,
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
