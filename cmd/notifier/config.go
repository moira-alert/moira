package main

import (
	"fmt"
	"time"

	"github.com/gosexy/to"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

type config struct {
	Redis       cmd.RedisConfig      `yaml:"redis"`
	Logger      cmd.LoggerConfig     `yaml:"log"`
	Notifier    notifierConfig       `yaml:"notifier"`
	Telemetry   cmd.TelemetryConfig  `yaml:"telemetry"`
	Remote      cmd.RemoteConfig     `yaml:"remote"`
	ImageStores cmd.ImageStoreConfig `yaml:"image_store"`
}

type notifierConfig struct {
	// Soft timeout to start retrying to send notification after single failed attempt
	SenderTimeout string `yaml:"sender_timeout"`
	// Hard timeout to stop retrying to send notification after multiple failed attempts
	ResendingTimeout string `yaml:"resending_timeout"`
	// Senders configuration section. See https://moira.readthedocs.io/en/latest/installation/configuration.html for more explanation
	Senders []map[string]string `yaml:"senders"`
	// Self state monitor configuration section. Note: No inner subscriptions is required. It's own notification mechanism will be used.
	SelfState selfStateConfig `yaml:"moira_selfstate"`
	// Web-UI uri prefix for trigger links in notifications. For example: with 'http://localhost' every notification will contain link like 'http://localhost/trigger/triggerId'
	FrontURI string `yaml:"front_uri"`
	// Timezone to use to convert ticks. Default is UTC. See https://golang.org/pkg/time/#LoadLocation for more details.
	Timezone string `yaml:"timezone"`
	// Format for email sender. Default is "15:04 02.01.2006". See https://golang.org/pkg/time/#Time.Format for more details about golang time formatting.
	DateTimeFormat string `yaml:"date_time_format"`
	// Amount of messages notifier reads from Redis per iteration. Use notifier.NotificationsLimitUnlimited for unlimited.
	ReadBatchSize int `yaml:"read_batch_size"`
}

type selfStateConfig struct {
	// If true, Self state monitor will be enabled
	Enabled bool `yaml:"enabled"`
	// If true, Self state monitor will check remote checker status
	RemoteTriggersEnabled bool `yaml:"remote_triggers_enabled"`
	// Max Redis disconnect delay to send alert when reached
	RedisDisconnectDelay string `yaml:"redis_disconect_delay"`
	// Max Filter metrics receive delay to send alert when reached
	LastMetricReceivedDelay string `yaml:"last_metric_received_delay"`
	// Max Checker checks perform delay to send alert when reached
	LastCheckDelay string `yaml:"last_check_delay"`
	// Max Remote triggers Checker checks perform delay to send alert when reached
	LastRemoteCheckDelay string `yaml:"last_remote_check_delay"`
	// Contact list for Self state monitor alerts
	Contacts []map[string]string `yaml:"contacts"`
	// Self state monitor alerting interval
	NoticeInterval string `yaml:"notice_interval"`
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			ConnectionLimit: 512,
			MetricsTTL:      "1h",
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
			FrontURI:      "http://localhost",
			Timezone:      "UTC",
			ReadBatchSize: int(notifier.NotificationsLimitUnlimited),
		},
		Telemetry: cmd.TelemetryConfig{
			Listen: ":8093",
			Graphite: cmd.GraphiteConfig{
				Enabled:      false,
				RuntimeStats: false,
				URI:          "localhost:2003",
				Prefix:       "DevOps.Moira",
				Interval:     "60s",
			},
			Pprof: cmd.ProfilerConfig{Enabled: false},
		},
		Remote: cmd.RemoteConfig{
			Timeout:    "60s",
			MetricsTTL: "24h",
		},
		ImageStores: cmd.ImageStoreConfig{},
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

	format := "15:04 02.01.2006"
	if err := checkDateTimeFormat(config.DateTimeFormat); err != nil {
		logger.Warningf("%v. Current time format: %v", err.Error(), time.Now().Format(format))
	} else {
		format = config.DateTimeFormat
		logger.Infof("Format '%v' parsed successfully. Current time format: %v", format, time.Now().Format(format))
	}

	readBatchSize := notifier.NotificationsLimitUnlimited
	if config.ReadBatchSize > 0 {
		readBatchSize = int64(config.ReadBatchSize)
	}
	if config.ReadBatchSize <= 0 && int64(config.ReadBatchSize) != notifier.NotificationsLimitUnlimited {
		logger.Warningf("Current read_batch_size is %d, but it should be > 0 or %v (unlimited), value ignored",
			config.ReadBatchSize, notifier.NotificationsLimitUnlimited)
	}
	logger.Infof("Current read_batch_size is %d", readBatchSize)

	return notifier.Config{
		SelfStateEnabled:  config.SelfState.Enabled,
		SelfStateContacts: config.SelfState.Contacts,
		SendingTimeout:    to.Duration(config.SenderTimeout),
		ResendingTimeout:  to.Duration(config.ResendingTimeout),
		Senders:           config.Senders,
		FrontURL:          config.FrontURI,
		Location:          location,
		DateTimeFormat:    format,
		ReadBatchSize:     readBatchSize,
	}
}

func checkDateTimeFormat(format string) error {
	fallbackTime := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	parsedTime, err := time.Parse(format, time.Now().Format(format))
	if err != nil || parsedTime == fallbackTime {
		return fmt.Errorf("could not parse date time format '%v', result: '%v', error: '%v'", format, parsedTime, err)
	}
	return nil
}

func (config *selfStateConfig) getSettings() selfstate.Config {
	return selfstate.Config{
		Enabled:                        config.Enabled,
		RedisDisconnectDelaySeconds:    int64(to.Duration(config.RedisDisconnectDelay).Seconds()),
		LastMetricReceivedDelaySeconds: int64(to.Duration(config.LastMetricReceivedDelay).Seconds()),
		LastCheckDelaySeconds:          int64(to.Duration(config.LastCheckDelay).Seconds()),
		LastRemoteCheckDelaySeconds:    int64(to.Duration(config.LastRemoteCheckDelay).Seconds()),
		Contacts:                       config.Contacts,
		NoticeIntervalSeconds:          int64(to.Duration(config.NoticeInterval).Seconds()),
	}
}
