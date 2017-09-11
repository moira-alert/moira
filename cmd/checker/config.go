package main

import (
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
	"menteslibres.net/gosexy/to"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Checker  checkerConfig      `yaml:"checker"`
}

type checkerConfig struct {
	NoDataCheckInterval  string `yaml:"nodata_check_interval"`
	CheckInterval        string `yaml:"check_interval"`
	MetricsTTL           int64  `yaml:"metrics_ttl"`
	StopCheckingInterval int64  `yaml:"stop_checking_interval"`
}

func (config *checkerConfig) getSettings() *checker.Config {
	return &checker.Config{
		MetricsTTL:           config.MetricsTTL,
		CheckInterval:        to.Duration(config.CheckInterval),
		NoDataCheckInterval:  to.Duration(config.NoDataCheckInterval),
		StopCheckingInterval: config.StopCheckingInterval,
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
			NoDataCheckInterval:  "60s0ms",
			CheckInterval:        "5s0ms",
			MetricsTTL:           3600,
			StopCheckingInterval: 30,
		},
		Graphite: cmd.GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s0ms",
		},
	}
}
