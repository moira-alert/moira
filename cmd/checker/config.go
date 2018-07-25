package main

import (
	"runtime"

	"github.com/gosexy/to"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis     cmd.RedisConfig    `yaml:"redis"`
	Graphite  cmd.GraphiteConfig `yaml:"graphite"`
	Logger    cmd.LoggerConfig   `yaml:"log"`
	Checker   checkerConfig      `yaml:"checker"`
	Pprof     cmd.ProfilerConfig `yaml:"pprof"`
	Migration migrationConfig    `yaml:"migration"`
}

type checkerConfig struct {
	// Period for every trigger to perform forced check on
	NoDataCheckInterval string `yaml:"nodata_check_interval"`
	// Period for every trigger to cancel forced check (earlier than 'NoDataCheckInterval') if no metrics were received
	StopCheckingInterval string `yaml:"stop_checking_interval"`
	// Min period to perform triggers re-check. Note: Reducing of this value leads to increasing of CPU and memory usage values
	CheckInterval string `yaml:"check_interval"`
	// Time interval to store metrics. Note: Increasing of this value leads to increasing of Redis memory consumption value
	MetricsTTL string `yaml:"metrics_ttl"`
	// Max concurrent checkers to run. Equals to the number of processor cores found on Moira host by default or when variable is defined as 0.
	MaxParallelChecks int `yaml:"max_parallel_checks"`
}

type migrationConfig struct {
	Enabled bool `yaml:"enabled"`
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
			LogLevel: "info",
		},
		Checker: checkerConfig{
			NoDataCheckInterval:  "60s",
			CheckInterval:        "5s",
			MetricsTTL:           "1h",
			StopCheckingInterval: "30s",
			MaxParallelChecks:    0,
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
		Migration: migrationConfig{
			Enabled: true,
		},
	}
}
