package main

import (
	"fmt"
	"time"

	"github.com/xiam/to"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

type config struct {
	Redis               cmd.RedisConfig               `yaml:"redis"`
	Logger              cmd.LoggerConfig              `yaml:"log"`
	Notifier            notifierConfig                `yaml:"notifier"`
	Telemetry           cmd.TelemetryConfig           `yaml:"telemetry"`
	Remotes             cmd.RemotesConfig             `yaml:",inline"`
	ImageStores         cmd.ImageStoreConfig          `yaml:"image_store"`
	NotificationHistory cmd.NotificationHistoryConfig `yaml:"notification_history"`
	Notification        cmd.NotificationConfig        `yaml:"notification"`
}

type entityLogConfig struct {
	ID    string `yaml:"id"`
	Level string `yaml:"level"`
}

type setLogLevelConfig struct {
	Contacts      []entityLogConfig `yaml:"contacts"`
	Subscriptions []entityLogConfig `yaml:"subscriptions"`
}

type notifierConfig struct {
	// Soft timeout to start retrying to send notification after single failed attempt
	SenderTimeout string `yaml:"sender_timeout"`
	// Hard timeout to stop retrying to send notification after multiple failed attempts
	ResendingTimeout string `yaml:"resending_timeout"`
	// Delay before performing one more send attempt
	ReschedulingDelay string `yaml:"rescheduling_delay"`
	// Senders configuration section. See https://moira.readthedocs.io/en/latest/installation/configuration.html for more explanation
	Senders []map[string]interface{} `yaml:"senders"`
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
	// Count available mute resend call, if more than set - you see error in logs
	MaxFailAttemptToSendAvailable int `yaml:"max_fail_attempt_to_send_available"`
	// Specify log level by entities
	SetLogLevel setLogLevelConfig `yaml:"set_log_level"`
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
	// Self state monitor check interval
	CheckInterval string `yaml:"check_interval"`
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Addrs:       "localhost:6379",
			MetricsTTL:  "1h",
			DialTimeout: "500ms",
		},
		Logger: cmd.LoggerConfig{
			LogFile:         "stdout",
			LogLevel:        "info",
			LogPrettyFormat: false,
		},
		NotificationHistory: cmd.NotificationHistoryConfig{
			NotificationHistoryTTL:        "48h",
			NotificationHistoryQueryLimit: int(notifier.NotificationsLimitUnlimited),
		},
		Notification: cmd.NotificationConfig{
			DelayedTime:               "60s",
			TransactionTimeout:        "100ms",
			TransactionMaxRetries:     10,
			TransactionHeuristicLimit: 10000,
			ResaveTime:                "30s",
		},
		Notifier: notifierConfig{
			SenderTimeout:     "10s",
			ResendingTimeout:  "1:00",
			ReschedulingDelay: "60s",
			SelfState: selfStateConfig{
				Enabled:                 false,
				RedisDisconnectDelay:    "30s",
				LastMetricReceivedDelay: "60s",
				LastCheckDelay:          "60s",
				NoticeInterval:          "300s",
			},
			FrontURI:                      "http://localhost",
			Timezone:                      "UTC",
			ReadBatchSize:                 int(notifier.NotificationsLimitUnlimited),
			MaxFailAttemptToSendAvailable: 3,
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
		Remotes:     cmd.RemotesConfig{},
		ImageStores: cmd.ImageStoreConfig{},
	}
}

func (config *notifierConfig) getSettings(logger moira.Logger) notifier.Config {
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		logger.Warning().
			String("timezone", config.Timezone).
			Error(err).
			Msg("Timezone load failed. Use UTC.")
		location, _ = time.LoadLocation("UTC")
	} else {
		logger.Info().
			String("timezone", config.Timezone).
			Msg("Timezone loaded")
	}

	format := "15:04 02.01.2006"
	if err := checkDateTimeFormat(config.DateTimeFormat); err != nil {
		logger.Warning().
			String("current_time_format", time.Now().Format(format)).
			Error(err).
			Msg("Failed to change time format")
	} else {
		format = config.DateTimeFormat
		logger.Info().
			String("format", format).
			String("current_time_format", time.Now().Format(format)).
			Msg("Format parsed successfully")
	}

	readBatchSize := notifier.NotificationsLimitUnlimited
	if config.ReadBatchSize > 0 {
		readBatchSize = int64(config.ReadBatchSize)
	}
	if config.ReadBatchSize <= 0 && int64(config.ReadBatchSize) != notifier.NotificationsLimitUnlimited {
		logger.Warning().
			Int("read_batch_size", config.ReadBatchSize).
			Int64("notification_limit_unlimited", notifier.NotificationsLimitUnlimited).
			Msg("Current config's read_batch_size is invalid, value ignored")
	}
	logger.Info().
		Int64("read_batch_size", readBatchSize).
		Msg("Current read_batch_size")

	contacts := map[string]string{}
	for _, v := range config.SetLogLevel.Contacts {
		contacts[v.ID] = v.Level
	}
	subscriptions := map[string]string{}
	for _, v := range config.SetLogLevel.Subscriptions {
		subscriptions[v.ID] = v.Level
	}
	logger.Info().
		Int("contacts_count", len(contacts)).
		Int("subscriptions_count", len(subscriptions)).
		Msg("Found dynamic log rules in config for some contacts and subscriptions")

	return notifier.Config{
		SelfStateEnabled:              config.SelfState.Enabled,
		SelfStateContacts:             config.SelfState.Contacts,
		SendingTimeout:                to.Duration(config.SenderTimeout),
		ResendingTimeout:              to.Duration(config.ResendingTimeout),
		ReschedulingDelay:             to.Duration(config.ReschedulingDelay),
		Senders:                       config.Senders,
		FrontURL:                      config.FrontURI,
		Location:                      location,
		DateTimeFormat:                format,
		ReadBatchSize:                 readBatchSize,
		MaxFailAttemptToSendAvailable: config.MaxFailAttemptToSendAvailable,
		LogContactsToLevel:            contacts,
		LogSubscriptionsToLevel:       subscriptions,
	}
}

func checkDateTimeFormat(format string) error {
	fallbackTime := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	parsedTime, err := time.Parse(format, time.Now().Format(format))
	if err != nil || parsedTime == fallbackTime {
		return fmt.Errorf("could not parse date time format '%v', result: '%v', error: '%w'", format, parsedTime, err)
	}
	return nil
}

func (config *selfStateConfig) getSettings() selfstate.Config {
	// 10 sec is default check value
	checkInterval := 10 * time.Second
	if config.CheckInterval != "" {
		checkInterval = to.Duration(config.CheckInterval)
	}

	return selfstate.Config{
		Enabled:                        config.Enabled,
		RedisDisconnectDelaySeconds:    int64(to.Duration(config.RedisDisconnectDelay).Seconds()),
		LastMetricReceivedDelaySeconds: int64(to.Duration(config.LastMetricReceivedDelay).Seconds()),
		LastCheckDelaySeconds:          int64(to.Duration(config.LastCheckDelay).Seconds()),
		LastRemoteCheckDelaySeconds:    int64(to.Duration(config.LastRemoteCheckDelay).Seconds()),
		CheckInterval:                  checkInterval,
		Contacts:                       config.Contacts,
		NoticeIntervalSeconds:          int64(to.Duration(config.NoticeInterval).Seconds()),
	}
}
