package selfstate

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

type HeartbeatsCfg struct {
	DatabaseCfg      heartbeat.DatabaseHeartbeaterConfig
	FilterCfg        heartbeat.FilterHeartbeaterConfig
	LocalCheckerCfg  heartbeat.LocalCheckerHeartbeaterConfig
	RemoteCheckerCfg heartbeat.RemoteCheckerHeartbeaterConfig
	NotifierCfg      heartbeat.NotifierHeartbeaterConfig
}

type MonitorBaseConfig struct {
	Enabled        bool
	HeartbeatsCfg  HeartbeatsCfg
	NoticeInterval time.Duration `validate:"required,gt=0"`
	CheckInterval  time.Duration `validate:"required,gt=0"`
}

type AdminMonitorConfig struct {
	MonitorBaseConfig

	AdminContacts []map[string]string `validate:"required,min=1"`
}

func (cfg AdminMonitorConfig) validate(senders map[string]bool) error {
	if !cfg.Enabled {
		return nil
	}

	validator := validator.New()
	if err := validator.Struct(cfg); err != nil {
		return err
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

type UserMonitorConfig struct {
	MonitorBaseConfig
}

func (cfg UserMonitorConfig) validate() error {
	if !cfg.Enabled {
		return nil
	}

	validator := validator.New()
	return validator.Struct(cfg)
}

type MonitorConfig struct {
	AdminCfg AdminMonitorConfig
	UserCfg  UserMonitorConfig
}

type Config struct {
	Monitor MonitorConfig
}

func (cfg *Config) Validate(senders map[string]bool) error {
	if err := cfg.Monitor.AdminCfg.validate(senders); err != nil {
		return fmt.Errorf("admin config validation error: %w", err)
	}

	if err := cfg.Monitor.UserCfg.validate(); err != nil {
		return fmt.Errorf("user config validation error: %w", err)
	}

	return nil
}
