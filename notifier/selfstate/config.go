package selfstate

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

// HeartbeatConfig represents a heartbeat-specific settings.
type HeartbeatConfig struct {
	SystemTags []string
}

// ChecksConfig represents a checks list.
type ChecksConfig struct {
	Database      HeartbeatConfig
	Filter        HeartbeatConfig
	LocalChecker  HeartbeatConfig
	RemoteChecker HeartbeatConfig
	Notifier      HeartbeatConfig
}

// Config is representation of self state worker settings like moira admins contacts and threshold values for checked services.
type Config struct {
	Enabled                        bool
	RedisDisconnectDelaySeconds    int64
	LastMetricReceivedDelaySeconds int64
	LastCheckDelaySeconds          int64
	LastRemoteCheckDelaySeconds    int64
	CheckInterval                  time.Duration
	UserNotificationsInterval      time.Duration
	Contacts                       []map[string]string
	Checks                         ChecksConfig
}

func (config *Config) checkConfig(senders map[string]bool) error {
	if !config.Enabled {
		return nil
	}

	if len(config.Contacts) < 1 {
		return fmt.Errorf("contacts must be specified")
	}

	for _, adminContact := range config.Contacts {
		if _, ok := senders[adminContact["type"]]; !ok {
			return fmt.Errorf("unknown contact type [%s]", adminContact["type"])
		}

		if adminContact["value"] == "" {
			return fmt.Errorf("value for [%s] must be present", adminContact["type"])
		}
	}

	return nil
}

func (checksConfig *ChecksConfig) GetUniqueSystemTags() []string {
	systemTags := make([]string, 0)
	systemTags = append(systemTags, checksConfig.Database.SystemTags...)
	systemTags = append(systemTags, checksConfig.Filter.SystemTags...)
	systemTags = append(systemTags, checksConfig.LocalChecker.SystemTags...)
	systemTags = append(systemTags, checksConfig.RemoteChecker.SystemTags...)
	systemTags = append(systemTags, checksConfig.Notifier.SystemTags...)

	return moira.GetUniqueValues(systemTags...)
}
