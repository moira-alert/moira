package main

import (
	"time"

	"menteslibres.net/gosexy/to"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Notifier notifierConfig     `yaml:"notifier"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
}

type notifierConfig struct {
	SenderTimeout    string              `yaml:"sender_timeout"`
	ResendingTimeout string              `yaml:"resending_timeout"`
	Senders          []map[string]string `yaml:"senders"`
	SelfState        selfStateConfig     `yaml:"moira_selfstate"`
	FrontURI         string              `yaml:"front_uri"`
	Timezone         string              `yaml:"timezone"`
}

type selfStateConfig struct {
	Enabled                 bool                `yaml:"enabled"`
	RedisDisconnectDelay    string              `yaml:"redis_disconect_delay"`
	LastMetricReceivedDelay string              `yaml:"last_metric_received_delay"`
	LastCheckDelay          string              `yaml:"last_check_delay"`
	Contacts                []map[string]string `yaml:"contacts"`
	NoticeInterval          string              `yaml:"notice_interval"`
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host: "localhost",
			Port: "6379",
			DBID: 0,
		},
		Graphite: cmd.GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s",
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "debug",
		},
		Notifier: notifierConfig{
			SenderTimeout:    "10s",
			ResendingTimeout: "24:00",
			SelfState: selfStateConfig{
				Enabled:                 false,
				RedisDisconnectDelay:    "30s",
				LastMetricReceivedDelay: "60s",
				LastCheckDelay:          "60s",
				NoticeInterval:          "300s",
			},
			FrontURI: "http://localhost",
			Timezone: "UTC",
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}

func (config *notifierConfig) getSettings(logger moira.Logger) notifier.Config {
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		logger.Warningf("Timezone '%s' load failed: %s. Use UTC.", config.Timezone, err.Error())
		location, _ = time.LoadLocation("UTC")
	} else {
		logger.Infof("Timezone '%s' loaded.", config.Timezone)
	}

	return notifier.Config{
		SendingTimeout:   to.Duration(config.SenderTimeout),
		ResendingTimeout: to.Duration(config.ResendingTimeout),
		Senders:          config.Senders,
		FrontURL:         config.FrontURI,
		Location:         location,
	}
}

func (config *selfStateConfig) getSettings() selfstate.Config {
	return selfstate.Config{
		Enabled:                        config.Enabled,
		RedisDisconnectDelaySeconds:    int64(to.Duration(config.RedisDisconnectDelay).Seconds()),
		LastMetricReceivedDelaySeconds: int64(to.Duration(config.LastMetricReceivedDelay).Seconds()),
		LastCheckDelaySeconds:          int64(to.Duration(config.LastCheckDelay).Seconds()),
		Contacts:                       config.Contacts,
		NoticeIntervalSeconds:          int64(to.Duration(config.NoticeInterval).Seconds()),
	}
}
