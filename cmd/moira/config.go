package main

import (
	"menteslibres.net/gosexy/to"

	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/cmd"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Checker  checkerConfig      `yaml:"checker"`
	API      apiConfig          `yaml:"api"`
	Cache    cacheConfig        `yaml:"cache"`
	Notifier notifierConfig     `yaml:"notifier"`
}

// API Config

type apiConfig struct {
	Enabled string `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

func (config *apiConfig) getSettings() *api.Config {
	return &api.Config{
		Enabled: cmd.ToBool(config.Enabled),
		Listen:  config.Listen,
	}
}

// Cahce Config

type cacheConfig struct {
	Enabled         string `yaml:"enabled"`
	Listen          string `yaml:"listen"`
	RetentionConfig string `yaml:"retention-config"`
}

func (config *cacheConfig) getSettings() *cache.Config {
	return &cache.Config{
		Enabled:         cmd.ToBool(config.Enabled),
		Listen:          config.Listen,
		RetentionConfig: config.RetentionConfig,
	}
}

// Checher Config

type checkerConfig struct {
	Enabled              string `yaml:"enabled"`
	NoDataCheckInterval  string `yaml:"nodata_check_interval"`
	CheckInterval        int64  `yaml:"check_interval"`
	MetricsTTL           int64  `yaml:"metrics_ttl"`
	StopCheckingInterval int64  `yaml:"stop_checking_interval"`
}

func (config *checkerConfig) getSettings() *checker.Config {
	return &checker.Config{
		MetricsTTL:           config.MetricsTTL,
		CheckInterval:        config.CheckInterval,
		NoDataCheckInterval:  to.Duration(config.NoDataCheckInterval),
		StopCheckingInterval: config.StopCheckingInterval,
	}
}

//  Notifier Config

type notifierConfig struct {
	Enabled          string              `yaml:"enabled"`
	SenderTimeout    string              `yaml:"sender_timeout"`
	ResendingTimeout string              `yaml:"resending_timeout"`
	Senders          []map[string]string `yaml:"senders"`
	SelfState        selfStateConfig     `yaml:"moira_selfstate"`
}

type frontConfig struct {
	URI string `yaml:"uri"`
}

type selfStateConfig struct {
	Enabled                 string              `yaml:"enabled"`
	RedisDisconnectDelay    int64               `yaml:"redis_disconect_delay"`
	LastMetricReceivedDelay int64               `yaml:"last_metric_received_delay"`
	LastCheckDelay          int64               `yaml:"last_check_delay"`
	Contacts                []map[string]string `yaml:"contacts"`
	NoticeInterval          int64               `yaml:"notice_interval"`
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
			LogLevel: "debug",
		},
		API: apiConfig{
			Enabled: "true",
			Listen:  ":8081",
		},
		Cache: cacheConfig{
			Enabled:         "true",
			Listen:          ":2003",
			RetentionConfig: "storage-schemas.conf",
		},
		Checker: checkerConfig{
			Enabled:              "true",
			NoDataCheckInterval:  "60s0ms",
			CheckInterval:        5,
			MetricsTTL:           3600,
			StopCheckingInterval: 30,
		},
		Notifier: notifierConfig{
			Enabled:          "true",
			SenderTimeout:    "10s0ms",
			ResendingTimeout: "24:00",
			SelfState: selfStateConfig{
				Enabled:                 "false",
				RedisDisconnectDelay:    30,
				LastMetricReceivedDelay: 60,
				LastCheckDelay:          60,
				NoticeInterval:          300,
			},
		},
		Graphite: cmd.GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s0ms",
		},
	}
}
