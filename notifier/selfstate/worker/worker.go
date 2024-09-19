package worker

import (
	"errors"
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
)

var _ SelfstateWorker = (*selfstateWorker)(nil)

type SelfstateWorker interface {
	Start()
	Stop() error
}

type selfstateWorker struct {
	monitors []monitor.Monitor
}

func NewSelfstateWorker(
	cfg selfstate.Config,
	logger moira.Logger,
	database moira.Database,
	notifier notifier.Notifier,
	clock moira.Clock,
) (*selfstateWorker, error) {
	if err := cfg.Validate(notifier.GetSenders()); err != nil {
		return nil, fmt.Errorf("selfstate worker validation error: %w", err)
	}

	adminMonitorEnabled := cfg.Monitor.AdminCfg.Enabled
	userMonitorEnabled := cfg.Monitor.UserCfg.Enabled

	monitors := make([]monitor.Monitor, 0)

	if adminMonitorEnabled {
		adminMonitor, err := monitor.NewForAdmin(
			cfg.Monitor.AdminCfg,
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
			cfg.Monitor.UserCfg,
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

	return &selfstateWorker{
		monitors: monitors,
	}, nil
}

func (selfstateWorker *selfstateWorker) Start() {
	for _, monitor := range selfstateWorker.monitors {
		monitor.Start()
	}
}

func (selfstateWorker *selfstateWorker) Stop() error {
	stopErrors := make([]error, 0)

	for _, monitor := range selfstateWorker.monitors {
		if err := monitor.Stop(); err != nil {
			stopErrors = append(stopErrors, err)
		}
	}

	return errors.Join(stopErrors...)
}
