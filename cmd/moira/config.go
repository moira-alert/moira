package main

import (
	"menteslibres.net/gosexy/to"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Checker  checkerConfig      `yaml:"checker"`
	API      apiConfig          `yaml:"api"`
	Filter   filterConfig       `yaml:"filter"`
	Notifier notifierConfig     `yaml:"notifier"`
	LogFile  string             `yaml:"log_file"`
	LogLevel string             `yaml:"log_level"`
}

// API Config

type apiConfig struct {
	Enabled  string `yaml:"enabled"`
	Listen   string `yaml:"listen"`
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

func (config *apiConfig) getSettings() *api.Config {
	return &api.Config{
		Enabled: cmd.ToBool(config.Enabled),
		Listen:  config.Listen,
	}
}

// Filter Config

type filterConfig struct {
	Enabled         string `yaml:"enabled"`
	Listen          string `yaml:"listen"`
	RetentionConfig string `yaml:"retention-config"`
	LogFile         string `yaml:"log_file"`
	LogLevel        string `yaml:"log_level"`
}

func (config *filterConfig) getSettings() *filter.Config {
	return &filter.Config{
		Enabled:         cmd.ToBool(config.Enabled),
		Listen:          config.Listen,
		RetentionConfig: config.RetentionConfig,
	}
}

// Checher Config

type checkerConfig struct {
	Enabled              string `yaml:"enabled"`
	NoDataCheckInterval  string `yaml:"nodata_check_interval"`
	CheckInterval        string `yaml:"check_interval"`
	MetricsTTL           int64  `yaml:"metrics_ttl"`
	StopCheckingInterval int64  `yaml:"stop_checking_interval"`
	LogFile              string `yaml:"log_file"`
	LogLevel             string `yaml:"log_level"`
}

func (config *checkerConfig) getSettings() *checker.Config {
	return &checker.Config{
		Enabled:              cmd.ToBool(config.Enabled),
		MetricsTTL:           config.MetricsTTL,
		CheckInterval:        to.Duration(config.CheckInterval),
		NoDataCheckInterval:  to.Duration(config.NoDataCheckInterval),
		StopCheckingInterval: config.StopCheckingInterval,
		LogFile:              config.LogFile,
		LogLevel:             config.LogLevel,
	}
}

//  Notifier Config

type notifierConfig struct {
	Enabled          string              `yaml:"enabled"`
	SenderTimeout    string              `yaml:"sender_timeout"`
	ResendingTimeout string              `yaml:"resending_timeout"`
	Senders          []map[string]string `yaml:"senders"`
	SelfState        selfStateConfig     `yaml:"moira_selfstate"`
	LogFile          string              `yaml:"log_file"`
	LogLevel         string              `yaml:"log_level"`
	FrontURL         string              `yaml:"front_uri"`
}

func (config *notifierConfig) getSettings() *notifier.Config {
	return &notifier.Config{
		Enabled:          cmd.ToBool(config.Enabled),
		SendingTimeout:   to.Duration(config.SenderTimeout),
		ResendingTimeout: to.Duration(config.ResendingTimeout),
		Senders:          config.Senders,
		LogFile:          config.LogFile,
		LogLevel:         config.LogLevel,
		FrontURL:         config.FrontURL,
	}
}

type selfStateConfig struct {
	Enabled                 string              `yaml:"enabled"`
	RedisDisconnectDelay    int64               `yaml:"redis_disconect_delay"`
	LastMetricReceivedDelay int64               `yaml:"last_metric_received_delay"`
	LastCheckDelay          int64               `yaml:"last_check_delay"`
	Contacts                []map[string]string `yaml:"contacts"`
	NoticeInterval          int64               `yaml:"notice_interval"`
}

func (config *selfStateConfig) getSettings() *selfstate.Config {
	return &selfstate.Config{
		Enabled:                 cmd.ToBool(config.Enabled),
		RedisDisconnectDelay:    config.RedisDisconnectDelay,
		LastMetricReceivedDelay: config.LastMetricReceivedDelay,
		LastCheckDelay:          config.LastCheckDelay,
		Contacts:                config.Contacts,
		NoticeInterval:          config.NoticeInterval,
	}
}

func getDefault() config {
	return config{
		LogFile:  "stdout",
		LogLevel: "debug",
		Redis: cmd.RedisConfig{
			Host: "redis",
			Port: "6379",
			DBID: 0,
		},
		API: apiConfig{
			Enabled:  "true",
			Listen:   ":8081",
			LogFile:  "stdout",
			LogLevel: "debug",
		},
		Filter: filterConfig{
			Enabled:         "true",
			Listen:          ":2003",
			RetentionConfig: "storage-schemas.conf",
			LogFile:         "stdout",
			LogLevel:        "debug",
		},
		Checker: checkerConfig{
			Enabled:              "true",
			NoDataCheckInterval:  "60s0ms",
			CheckInterval:        "5s0ms",
			MetricsTTL:           3600,
			StopCheckingInterval: 30,
			LogFile:              "stdout",
			LogLevel:             "debug",
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
			FrontURL: "https:// moira.example.com",
			LogFile:  "stdout",
			LogLevel: "debug",
		},
		Graphite: cmd.GraphiteConfig{
			Enabled:  "false",
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s0ms",
		},
	}
}
