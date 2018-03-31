package main

import (
	"time"

	"github.com/gosexy/to"

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
	SenderTimeout    string              `yaml:"sender_timeout"`    // Soft timeout to start retrying to send notification after single failed attempt
	ResendingTimeout string              `yaml:"resending_timeout"` // Hard timeout to stop retrying to send notification after multiple failed attempts
	Senders          []map[string]string `yaml:"senders"`           // Senders configuration section. See https://moira.readthedocs.io/en/latest/installation/configuration.html for more explanation
	SelfState        selfStateConfig     `yaml:"moira_selfstate"`   // Self state monitor configuration section. Note: No inner subscriptions is required. It's own notification mechanism will be used.
	FrontURI         string              `yaml:"front_uri"`         // Web-UI uri prefix for trigger links in notifications. For example: with 'http://localhost' every notification will contain link like 'http://localhost/trigger/triggerId'
	Timezone         string              `yaml:"timezone"`          // Timezone to use to convert ticks. Default is UTC. See https://golang.org/pkg/time/#LoadLocation for more details.
}

type selfStateConfig struct {
	Enabled                 bool                `yaml:"enabled"`                    // If true, Self state monitor will be enabled.
	RedisDisconnectDelay    string              `yaml:"redis_disconect_delay"`      // Max Redis disconnect delay to send alert when reached
	LastMetricReceivedDelay string              `yaml:"last_metric_received_delay"` // Max Filter metrics receive delay to send alert when reached
	LastCheckDelay          string              `yaml:"last_check_delay"`           // Max Checker checks perform delay to send alert when reached
	Contacts                []map[string]string `yaml:"contacts"`                   // Contact list for Self state monitor alerts
	NoticeInterval          string              `yaml:"notice_interval"`            // Self state monitor alerting interval
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
			LogLevel: "info",
		},
		Notifier: notifierConfig{
			SenderTimeout:    "10s",
			ResendingTimeout: "1:00",
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
