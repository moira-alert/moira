package selfstate

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

// HeartbeatConfig represents a heartbeat-specific settings.
type HeartbeatConfig struct {
	SystemTags []string
}

// NotifierHeartbeatConfig represents a heartbeat-specific settings.
type NotifierHeartbeatConfig struct {
	AnyClusterSourceTags      []string
	LocalClusterSourceTags    []string
	TagPrefixForClusterSource string
}

// ChecksConfig represents a checks list.
type ChecksConfig struct {
	Database      HeartbeatConfig
	Filter        HeartbeatConfig
	LocalChecker  HeartbeatConfig
	RemoteChecker HeartbeatConfig
	Notifier      NotifierHeartbeatConfig
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
	FrontURL                       string
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

// GetUniqueSystemTags Value of this function could have been cached, but cache should not be a part of ChecksConfig structure.
// It could have been cached at place of usage, which would be more complicated.
// This function is not very compute/allocation heavy, so we've decided to omit caching for now.
func (checksConfig *ChecksConfig) GetUniqueSystemTags(clusterList moira.ClusterList) []string {
	systemTags := make([]string, 0)
	systemTags = append(systemTags, checksConfig.Database.SystemTags...)
	systemTags = append(systemTags, checksConfig.Filter.SystemTags...)
	systemTags = append(systemTags, checksConfig.LocalChecker.SystemTags...)
	systemTags = append(systemTags, checksConfig.RemoteChecker.SystemTags...)

	for _, key := range clusterList {
		tags := heartbeat.MakeNotifierTags(
			checksConfig.Notifier.AnyClusterSourceTags,
			checksConfig.Notifier.TagPrefixForClusterSource,
			checksConfig.Notifier.LocalClusterSourceTags,
			key,
		)
		systemTags = append(systemTags, tags...)
	}

	return moira.GetUniqueValues(systemTags...)
}
