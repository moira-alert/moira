package controller

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	w "github.com/moira-alert/moira/worker"
	"gopkg.in/tomb.v2"
)

// Verify that monitor matches the Monitor interface.
var _ Controller = (*controller)(nil)

const (
	name     = "Moira Selfstate Controller"
	lockName = "moira-selfstate-controller"
	lockTTL  = 15 * time.Second
)

// Controller interface that defines the methods of the selfstate controller.
type Controller interface {
	Start()
	Stop() error
}

// ControllerConfig defines the selfstate controller configuration.
type ControllerConfig struct {
	Enabled         bool
	HeartbeatersCfg heartbeat.HeartbeatersConfig `validate:"required_if=Enabled true"`
	CheckInterval   time.Duration                `validate:"required_if=Enabled true,gte=0"`
}

type controller struct {
	cfg          ControllerConfig
	tomb         tomb.Tomb
	logger       moira.Logger
	database     moira.Database
	heartbeaters []heartbeat.Heartbeater
}

// NewController is a function to create a new selfstate controller.
func NewController(
	cfg ControllerConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
) (*controller, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("controller configuration error: %w", err)
	}

	heartbeaters := createHeartbeaters(cfg.HeartbeatersCfg, logger, database, clock)

	return &controller{
		cfg:          cfg,
		logger:       logger,
		database:     database,
		heartbeaters: heartbeaters,
	}, nil
}

func createHeartbeaters(
	heartbeatersCfg heartbeat.HeartbeatersConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
) []heartbeat.Heartbeater { //nolint:gocyclo
	hearbeaterBase := heartbeat.NewHeartbeaterBase(logger, database, clock)

	heartbeaters := make([]heartbeat.Heartbeater, 0)

	if heartbeatersCfg.DatabaseCfg.Enabled && heartbeatersCfg.DatabaseCfg.NeedTurnOffNotifier {
		databaseHeartbeater, err := heartbeat.NewDatabaseHeartbeater(heartbeatersCfg.DatabaseCfg, hearbeaterBase)
		if err != nil {
			logger.Error().
				Error(err).
				String("heartbeater", string(databaseHeartbeater.Type())).
				Msg("Failed to create a new database heartbeater")
		} else {
			heartbeaters = append(heartbeaters, databaseHeartbeater)
		}
	}

	if heartbeatersCfg.FilterCfg.Enabled && heartbeatersCfg.FilterCfg.NeedTurnOffNotifier {
		filterHeartbeater, err := heartbeat.NewFilterHeartbeater(heartbeatersCfg.FilterCfg, hearbeaterBase)
		if err != nil {
			logger.Error().
				Error(err).
				String("heartbeater", string(filterHeartbeater.Type())).
				Msg("Failed to create a new filter heartbeater")
		} else {
			heartbeaters = append(heartbeaters, filterHeartbeater)
		}
	}

	if heartbeatersCfg.LocalCheckerCfg.Enabled && heartbeatersCfg.LocalCheckerCfg.NeedTurnOffNotifier {
		localCheckerHeartbeater, err := heartbeat.NewLocalCheckerHeartbeater(heartbeatersCfg.LocalCheckerCfg, hearbeaterBase)
		if err != nil {
			logger.Error().
				Error(err).
				String("heartbeater", string(localCheckerHeartbeater.Type())).
				Msg("Failed to create a new local checker heartbeater")
		} else {
			heartbeaters = append(heartbeaters, localCheckerHeartbeater)
		}
	}

	if heartbeatersCfg.RemoteCheckerCfg.Enabled && heartbeatersCfg.RemoteCheckerCfg.NeedTurnOffNotifier {
		remoteCheckerHeartbeater, err := heartbeat.NewRemoteCheckerHeartbeater(heartbeatersCfg.RemoteCheckerCfg, hearbeaterBase)
		if err != nil {
			logger.Error().
				Error(err).
				String("heartbeater", string(remoteCheckerHeartbeater.Type())).
				Msg("Failed to create a new remote checker heartbeater")
		} else {
			heartbeaters = append(heartbeaters, remoteCheckerHeartbeater)
		}
	}

	if heartbeatersCfg.NotifierCfg.Enabled && heartbeatersCfg.NotifierCfg.NeedTurnOffNotifier {
		notifierHeartbeater, err := heartbeat.NewNotifierHeartbeater(heartbeatersCfg.NotifierCfg, hearbeaterBase)
		if err != nil {
			logger.Error().
				Error(err).
				String("heartbeater", string(notifierHeartbeater.Type())).
				Msg("Failed to create a new notifier heartbeater")
		} else {
			heartbeaters = append(heartbeaters, notifierHeartbeater)
		}
	}

	return heartbeaters
}

// Start is the method to start the selfstate controller.
func (c *controller) Start() {
	c.tomb.Go(func() error {
		w.NewWorker(
			name,
			c.logger,
			c.database.NewLock(lockName, lockTTL),
			c.selfstateCheck,
		).Run(nil)
		return nil
	})
}

func (c *controller) selfstateCheck(stop <-chan struct{}) error {
	c.logger.Info().Msg(fmt.Sprintf("%s started", name))

	checkTicker := time.NewTicker(c.cfg.CheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			c.logger.Info().Msg(fmt.Sprintf("%s stopped", name))
			return nil
		case <-checkTicker.C:
			c.logger.Debug().Msg(fmt.Sprintf("%s selfstate check", name))

			c.checkHeartbeats()
		}
	}
}

func (c *controller) checkHeartbeats() {
	for _, heartbeater := range c.heartbeaters {
		heartbeatState, err := heartbeater.Check()
		if err != nil {
			c.logger.Error().
				Error(err).
				String("name", name).
				String("heartbeater", string(heartbeater.Type())).
				Msg("Heartbeat check failed")
		}

		if heartbeatState == heartbeat.StateError {
			if err = c.database.SetNotifierState(moira.SelfStateERROR); err != nil {
				c.logger.Error().
					Error(err).
					String("name", name).
					String("heartbeater", string(heartbeater.Type())).
					Msg("Failed to set notifier state to error")
			}
			break
		}
	}
}

// Stop is a method to stop the selfstate controller.
func (c *controller) Stop() error {
	c.tomb.Kill(nil)
	return c.tomb.Wait()
}
