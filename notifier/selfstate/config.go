package selfstate

import (
	"fmt"
)

//Config is representation of self state worker settings like moira admins contacts and threshold values for checked services
type Config struct {
	Enabled                 bool
	RedisDisconnectDelay    int64
	LastMetricReceivedDelay int64
	LastCheckDelay          int64
	Contacts                []map[string]string
	NoticeInterval          int64
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
