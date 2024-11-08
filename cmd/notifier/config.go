package main

import (
	"fmt"
	"time"

	"github.com/xiam/to"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
	"github.com/moira-alert/moira/notifier/selfstate/controller"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
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
	// Selfstate monitor configuration section. Note: No inner subscriptions is required. It's own notification mechanism will be used.
	Selfstate selfstateConfig `yaml:"moira_selfstate"`
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

// heartbeaterAlertConfig sets the configuration for the alert of a particular heartbeater.
type heartbeaterAlertConfig struct {
	Name string `yaml:"name"`
	Desc string `yaml:"desc"`
}

// heartbeaterBaseConfig sets the base configuration of heartbeater.
type heartbeaterBaseConfig struct {
	Enabled             bool `yaml:"enabled"`
	NeedTurnOffNotifier bool `yaml:"need_turn_off_notifier"`

	AlertCfg heartbeaterAlertConfig `yaml:"alert"`
}

func (cfg heartbeaterBaseConfig) getSettings() heartbeat.HeartbeaterBaseConfig {
	return heartbeat.HeartbeaterBaseConfig{
		Enabled:             cfg.Enabled,
		NeedTurnOffNotifier: cfg.NeedTurnOffNotifier,

		AlertCfg: heartbeat.AlertConfig{
			Name: cfg.AlertCfg.Name,
			Desc: cfg.AlertCfg.Desc,
		},
	}
}

// databaseHeartbeaterConfig defines the database heartbeater configuration.
type databaseHeartbeaterConfig struct {
	heartbeaterBaseConfig `yaml:",inline"`

	RedisDisconnectDelay string `yaml:"redis_disconnect_delay"`
}

// filterHeartbeaterConfig defines the filter heartbeater configuration.
type filterHeartbeaterConfig struct {
	heartbeaterBaseConfig `yaml:",inline"`

	MetricReceivedDelay string `yaml:"last_metric_received_delay"`
}

// localCheckerHeartbeaterConfig defines the local checker heartbeater configuration.
type localCheckerHeartbeaterConfig struct {
	heartbeaterBaseConfig `yaml:",inline"`

	LocalCheckDelay string `yaml:"last_check_delay"`
}

// remoteCheckerHeartbeaterConfig defines the remote checker heartbeater configuration.
type remoteCheckerHeartbeaterConfig struct {
	heartbeaterBaseConfig `yaml:",inline"`

	RemoteCheckDelay string `yaml:"last_remote_check_delay"`
}

// notifierHeartbeaterConfig defines the notifier heartbeater configuration.
type notifierHeartbeaterConfig struct {
	heartbeaterBaseConfig `yaml:",inline"`
}

// heartbeatsConfig defines the configuration of heartbeaters.
type heartbeatsConfig struct {
	DatabaseCfg      databaseHeartbeaterConfig      `yaml:"database"`
	FilterCfg        filterHeartbeaterConfig        `yaml:"filter"`
	LocalCheckerCfg  localCheckerHeartbeaterConfig  `yaml:"local_checker"`
	RemoteCheckerCfg remoteCheckerHeartbeaterConfig `yaml:"remote_checker"`
	NotifierCfg      notifierHeartbeaterConfig      `yaml:"notifier"`
}

func (cfg heartbeatsConfig) getSettings() heartbeat.HeartbeatersConfig {
	return heartbeat.HeartbeatersConfig{
		DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
			HeartbeaterBaseConfig: cfg.DatabaseCfg.heartbeaterBaseConfig.getSettings(),
			RedisDisconnectDelay:  to.Duration(cfg.DatabaseCfg.RedisDisconnectDelay),
		},
		FilterCfg: heartbeat.FilterHeartbeaterConfig{
			HeartbeaterBaseConfig: cfg.FilterCfg.heartbeaterBaseConfig.getSettings(),
			MetricReceivedDelay:   to.Duration(cfg.FilterCfg.MetricReceivedDelay),
		},
		LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: cfg.LocalCheckerCfg.heartbeaterBaseConfig.getSettings(),
			LocalCheckDelay:       to.Duration(cfg.LocalCheckerCfg.LocalCheckDelay),
		},
		RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: cfg.RemoteCheckerCfg.heartbeaterBaseConfig.getSettings(),
			RemoteCheckDelay:      to.Duration(cfg.RemoteCheckerCfg.RemoteCheckDelay),
		},
		NotifierCfg: heartbeat.NotifierHeartbeaterConfig{
			HeartbeaterBaseConfig: cfg.NotifierCfg.heartbeaterBaseConfig.getSettings(),
		},
	}
}

// monitorBaseConfig defines the basic configuration of the monitor.
type monitorBaseConfig struct {
	Enabled        bool             `yaml:"enabled"`
	HearbeatersCfg heartbeatsConfig `yaml:"heartbeaters"`
	NoticeInterval string           `yaml:"notice_interval"`
	CheckInterval  string           `yaml:"check_interval"`
}

// adminMonitorConfig defines the configuration for the admin monitor.
type adminMonitorConfig struct {
	monitorBaseConfig `yaml:",inline"`

	AdminContacts []map[string]string `yaml:"contacts"`
}

// userMonitorConfig defines the configuration for the user monitor.
type userMonitorConfig struct {
	monitorBaseConfig `yaml:",inline"`
}

// monitorConfig defines the configuration for all monitors.
type monitorConfig struct {
	AdminCfg adminMonitorConfig `yaml:"admin"`
	UserCfg  userMonitorConfig  `yaml:"user"`
}

// controllerConfig defines the selfstate controller configuration.
type controllerConfig struct {
	Enabled        bool             `yaml:"enabled"`
	HearbeatersCfg heartbeatsConfig `yaml:"heartbeaters"`
	CheckInterval  string           `yaml:"check_interval"`
}

// selfstateConfig defines the configuration of the selfstate worker.
type selfstateConfig struct {
	Enabled       bool             `yaml:"enabled"`
	MonitorCfg    monitorConfig    `yaml:"monitor"`
	ControllerCfg controllerConfig `yaml:"controller"`
}

func (cfg *selfstateConfig) getSettings() selfstate.Config {
	return selfstate.Config{
		Enabled: cfg.Enabled,
		MonitorCfg: selfstate.MonitorConfig{
			AdminCfg: monitor.AdminMonitorConfig{
				MonitorBaseConfig: monitor.MonitorBaseConfig{
					Enabled:         cfg.MonitorCfg.AdminCfg.Enabled,
					HeartbeatersCfg: cfg.MonitorCfg.AdminCfg.HearbeatersCfg.getSettings(),
					NoticeInterval:  to.Duration(cfg.MonitorCfg.AdminCfg.NoticeInterval),
					CheckInterval:   to.Duration(cfg.MonitorCfg.AdminCfg.CheckInterval),
				},
				AdminContacts: cfg.MonitorCfg.AdminCfg.AdminContacts,
			},
			UserCfg: monitor.UserMonitorConfig{
				MonitorBaseConfig: monitor.MonitorBaseConfig{
					Enabled:         cfg.MonitorCfg.UserCfg.Enabled,
					HeartbeatersCfg: cfg.MonitorCfg.UserCfg.HearbeatersCfg.getSettings(),
					NoticeInterval:  to.Duration(cfg.MonitorCfg.UserCfg.NoticeInterval),
					CheckInterval:   to.Duration(cfg.MonitorCfg.UserCfg.CheckInterval),
				},
			},
		},
		ControllerCfg: controller.ControllerConfig{
			Enabled:         cfg.ControllerCfg.Enabled,
			HeartbeatersCfg: cfg.ControllerCfg.HearbeatersCfg.getSettings(),
			CheckInterval:   to.Duration(cfg.ControllerCfg.CheckInterval),
		},
	}
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
			NotificationHistoryTTL: "48h",
		},
		Notification: cmd.NotificationConfig{
			DelayedTime:               "50s",
			TransactionTimeout:        "100ms",
			TransactionMaxRetries:     10,
			TransactionHeuristicLimit: 10000,
			ResaveTime:                "30s",
		},
		Notifier: notifierConfig{
			SenderTimeout:     "10s",
			ResendingTimeout:  "1:00",
			ReschedulingDelay: "60s",
			Selfstate: selfstateConfig{
				Enabled: false,
				MonitorCfg: monitorConfig{
					AdminCfg: adminMonitorConfig{
						monitorBaseConfig: monitorBaseConfig{
							Enabled: false,
							HearbeatersCfg: heartbeatsConfig{
								DatabaseCfg: databaseHeartbeaterConfig{
									RedisDisconnectDelay: "30s",
								},
								FilterCfg: filterHeartbeaterConfig{
									MetricReceivedDelay: "60s",
								},
								LocalCheckerCfg: localCheckerHeartbeaterConfig{
									LocalCheckDelay: "60s",
								},
								RemoteCheckerCfg: remoteCheckerHeartbeaterConfig{
									RemoteCheckDelay: "300s",
								},
								NotifierCfg: notifierHeartbeaterConfig{},
							},
						},
					},
					UserCfg: userMonitorConfig{
						monitorBaseConfig: monitorBaseConfig{
							Enabled: false,
							HearbeatersCfg: heartbeatsConfig{
								DatabaseCfg: databaseHeartbeaterConfig{
									RedisDisconnectDelay: "30s",
								},
								FilterCfg: filterHeartbeaterConfig{
									MetricReceivedDelay: "60s",
								},
								LocalCheckerCfg: localCheckerHeartbeaterConfig{
									LocalCheckDelay: "60s",
								},
								RemoteCheckerCfg: remoteCheckerHeartbeaterConfig{
									RemoteCheckDelay: "300s",
								},
								NotifierCfg: notifierHeartbeaterConfig{},
							},
						},
					},
				},
				ControllerCfg: controllerConfig{
					Enabled:       false,
					CheckInterval: "10s",
				},
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
		SelfstateEnabled:              config.Selfstate.Enabled,
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
