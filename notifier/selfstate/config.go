package selfstate

import (
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
)

type MonitorConfig struct {
	UserCfg  monitor.UserMonitorConfig
	AdminCfg monitor.AdminMonitorConfig
}

type Config struct {
	Enabled    bool
	MonitorCfg MonitorConfig
}
