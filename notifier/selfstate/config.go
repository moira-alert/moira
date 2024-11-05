package selfstate

import (
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
)

// MonitorConfig sets the configurations of all monitors.
type MonitorConfig struct {
	UserCfg  monitor.UserMonitorConfig
	AdminCfg monitor.AdminMonitorConfig
}

// Config sets the configuration of the selfstate worker.
type Config struct {
	Enabled    bool
	MonitorCfg MonitorConfig
}
