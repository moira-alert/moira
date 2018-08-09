package selfstate

import (
	"fmt"
)

// Config is representation of self state worker settings like moira admins contacts and threshold values for checked services
type Config struct {
	Enabled                        bool
	RedisDisconnectDelaySeconds    int64
	LastMetricReceivedDelaySeconds int64
	LastCheckDelaySeconds          int64
	LastRemoteCheckDelaySeconds    int64
	RemoteTriggersEnabled          bool
	Contacts                       []map[string]string
	NoticeIntervalSeconds          int64
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
			return fmt.Errorf("Unknown contact type [%s]", adminContact["type"])
		}
		if adminContact["value"] == "" {
			return fmt.Errorf("Value for [%s] must be present", adminContact["type"])
		}
	}
	return nil
}
