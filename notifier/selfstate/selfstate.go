package selfstate

import (
	"errors"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
)

// Verify that selfstateWorker matches the SelfstateWorker interface.
var _ SelfstateWorker = (*selfstateWorker)(nil)

// SelfstateWorker interface, which defines methods for starting and stopping the selfstate worker.
type SelfstateWorker interface {
	Start()
	Stop() error
}

type selfstateWorker struct {
	monitors []monitor.Monitor
}

// NewSelfstateWorker is a method to create a new selfstate worker.
func NewSelfstateWorker(
	cfg Config,
	logger moira.Logger,
	database moira.Database,
	notifier notifier.Notifier,
	clock moira.Clock,
) (*selfstateWorker, error) {
	monitors := createMonitors(cfg.MonitorCfg, logger, database, clock, notifier)

	return &selfstateWorker{
		monitors: monitors,
	}, nil
}

func createMonitors(
	monitorCfg MonitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
) []monitor.Monitor {
	adminMonitorEnabled := monitorCfg.AdminCfg.Enabled
	userMonitorEnabled := monitorCfg.UserCfg.Enabled

	monitors := make([]monitor.Monitor, 0)

	if adminMonitorEnabled {
		adminMonitor, err := monitor.NewForAdmin(
			monitorCfg.AdminCfg,
			logger,
			database,
			clock,
			notifier,
		)
		if err != nil {
			logger.Error().
				Error(err).
				Msg("Failed to create a new admin monitor")
		} else {
			monitors = append(monitors, adminMonitor)
		}
	}

	if userMonitorEnabled {
		userMonitor, err := monitor.NewForUser(
			monitorCfg.UserCfg,
			logger,
			database,
			clock,
			notifier,
		)
		if err != nil {
			logger.Error().
				Error(err).
				Msg("Failed to create a new user monitor")
		} else {
			monitors = append(monitors, userMonitor)
		}
	}

	return monitors
}

// Start is a method to start a selfstate worker.
func (selfstateWorker *selfstateWorker) Start() {
	for _, monitor := range selfstateWorker.monitors {
		monitor.Start()
	}
}

// Stop is a method for stopping a selfstate worker.
func (selfstateWorker *selfstateWorker) Stop() error {
	stopErrors := make([]error, 0)

	for _, monitor := range selfstateWorker.monitors {
		if err := monitor.Stop(); err != nil {
			stopErrors = append(stopErrors, err)
		}
	}

	return errors.Join(stopErrors...)
}
