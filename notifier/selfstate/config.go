package selfstate

import (
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

var ErrNoAdminContacts = errors.New("admin contacts must be specified")

type HeartbeatsCfg struct {
	DatabaseCfg      heartbeat.DatabaseHeartbeaterConfig
	FilterCfg        heartbeat.FilterHeartbeaterConfig
	LocalCheckerCfg  heartbeat.LocalCheckerHeartbeaterConfig
	RemoteCheckerCfg heartbeat.RemoteCheckerHeartbeaterConfig
	NotifierCfg      heartbeat.NotifierHeartbeaterConfig
}

type selfstateBaseConfig struct {
	Enabled        bool
	HeartbeatsCfg  HeartbeatsCfg
	NoticeInterval time.Duration
	CheckInterval  time.Duration
}

type AdminConfig struct {
	selfstateBaseConfig

	AdminContacts []map[string]string
}

func (cfg AdminConfig) validate(senders map[string]bool) error {
	if !cfg.Enabled {
		return nil
	}

	if len(cfg.AdminContacts) < 1 {
		return ErrNoAdminContacts
	}

	for _, contact := range cfg.AdminContacts {
		if _, ok := senders[contact["type"]]; !ok {
			return fmt.Errorf("unknown contact type [%s]", contact["type"])
		}

		if contact["value"] == "" {
			return fmt.Errorf("value for [%s] must be present", contact["type"])
		}
	}

	return nil
}

type UserConfig struct {
	selfstateBaseConfig
}

type Config struct {
	AdminCfg AdminConfig
	UserCfg  UserConfig
}

// // Config is representation of self state worker settings like moira admins contacts and threshold values for checked services.
// type Config struct {
// 	Enabled                        bool
// 	RedisDisconnectDelaySeconds    int64
// 	LastMetricReceivedDelaySeconds int64
// 	LastCheckDelaySeconds          int64
// 	LastRemoteCheckDelaySeconds    int64
// 	NoticeIntervalSeconds          int64
// 	CheckInterval                  time.Duration
// 	Contacts                       []map[string]string
// }

func (cfg *Config) checkConfig(senders map[string]bool) error {
	if err := cfg.AdminCfg.validate(senders); err != nil {
		return fmt.Errorf("failed to validate admin config: %w", err)
	}

	return nil
}
