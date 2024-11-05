package monitor

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	w "github.com/moira-alert/moira/worker"
	"gopkg.in/tomb.v2"
)

var (
	okValue           = 0.0
	errorValue        = 1.0
	triggerErrorValue = 1.0

	_ Monitor = (*monitor)(nil)
)

type MonitorBaseConfig struct {
	Enabled         bool
	HeartbeatersCfg heartbeat.HeartbeatersConfig `validate:"required_if=Enabled true"`
	NoticeInterval  time.Duration                `validate:"required_if=Enabled true,gte=0"`
	CheckInterval   time.Duration                `validate:"required_if=Enabled true,gte=0"`
}

type hearbeatInfo struct {
	lastAlertTime  time.Time
	lastCheckState heartbeat.State
}

type monitorConfig struct {
	Name           string        `validate:"required"`
	LockName       string        `validate:"required"`
	LockTTL        time.Duration `validate:"required,gt=0"`
	NoticeInterval time.Duration `validate:"required,gt=0"`
	CheckInterval  time.Duration `validate:"required,gt=0"`
}

type Monitor interface {
	Start()
	Stop() error
}

type monitor struct {
	cfg               monitorConfig
	logger            moira.Logger
	database          moira.Database
	notifier          notifier.Notifier
	tomb              tomb.Tomb
	heartbeaters      []heartbeat.Heartbeater
	clock             moira.Clock
	heartbeatsInfo    map[datatypes.HeartbeatType]*hearbeatInfo
	sendNotifications func(pkgs []notifier.NotificationPackage) error
}

func newMonitor(
	cfg monitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
	heartbeaters []heartbeat.Heartbeater,
	sendNotifications func(pkgs []notifier.NotificationPackage) error,
) (*monitor, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("monitor configuration error: %w", err)
	}

	hearbeatersInfo := make(map[datatypes.HeartbeatType]*hearbeatInfo, len(heartbeaters))
	for _, heartbeater := range heartbeaters {
		hearbeatersInfo[heartbeater.Type()] = &hearbeatInfo{
			lastCheckState: heartbeat.StateOK,
		}
	}

	return &monitor{
		cfg:               cfg,
		logger:            logger,
		database:          database,
		notifier:          notifier,
		heartbeaters:      heartbeaters,
		clock:             clock,
		heartbeatsInfo:    hearbeatersInfo,
		sendNotifications: sendNotifications,
	}, nil
}

func createHearbeaters(
	heartbeatersCfg heartbeat.HeartbeatersConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
) []heartbeat.Heartbeater {
	hearbeaterBase := heartbeat.NewHeartbeaterBase(logger, database, clock)

	heartbeaters := make([]heartbeat.Heartbeater, 0)

	if heartbeatersCfg.DatabaseCfg.Enabled {
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

	if heartbeatersCfg.FilterCfg.Enabled {
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

	if heartbeatersCfg.LocalCheckerCfg.Enabled {
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

	if heartbeatersCfg.RemoteCheckerCfg.Enabled {
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

	if heartbeatersCfg.NotifierCfg.Enabled {
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

func (m *monitor) Start() {
	m.tomb.Go(func() error {
		w.NewWorker(
			m.cfg.Name,
			m.logger,
			m.database.NewLock(m.cfg.LockName, m.cfg.LockTTL),
			m.selfstateCheck,
		).Run(nil)
		return nil
	})
}

func (m *monitor) selfstateCheck(stop <-chan struct{}) error {
	m.logger.Info().Msg(fmt.Sprintf("%s started", m.cfg.Name))

	checkTicker := time.NewTicker(m.cfg.CheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			m.logger.Info().Msg(fmt.Sprintf("%s stopped", m.cfg.Name))
			return nil
		case <-checkTicker.C:
			m.logger.Debug().Msg(fmt.Sprintf("%s selfstate check", m.cfg.Name))

			m.check()
		}
	}
}

func (m *monitor) check() {
	pkgs := m.checkHeartbeats()
	if len(pkgs) > 0 {
		if err := m.sendNotifications(pkgs); err != nil {
			m.logger.Error().
				Error(err).
				String("type", m.cfg.Name).
				Interface("notification_packages", pkgs).
				Msg("Failed to send heartbeats notifications")
		}
	}
}

func (m *monitor) checkHeartbeats() []notifier.NotificationPackage {
	pkgs := make([]notifier.NotificationPackage, 0)

	for _, heartbeater := range m.heartbeaters {
		heartbeatState, err := heartbeater.Check()
		if err != nil {
			m.logger.Error().
				Error(err).
				String("name", m.cfg.Name).
				String("heartbeater", string(heartbeater.Type())).
				Msg("Heartbeat check failed")
		}

		m.logger.Debug().
			String("name", m.cfg.Name).
			String("heartbeater", string(heartbeater.Type())).
			String("state", string(heartbeatState)).
			Msg("Check heartbeat")

		pkg := m.generateHeartbeatNotificationPackage(heartbeater, heartbeatState)
		if pkg != nil {
			pkgs = append(pkgs, *pkg)
		}

		m.heartbeatsInfo[heartbeater.Type()].lastCheckState = heartbeatState
	}

	return pkgs
}

func (m *monitor) generateHeartbeatNotificationPackage(heartbeater heartbeat.Heartbeater, heartbeatState heartbeat.State) *notifier.NotificationPackage {
	heartbeatInfo := m.heartbeatsInfo[heartbeater.Type()]

	now := m.clock.NowUTC()

	isDegraded := heartbeatInfo.lastCheckState.IsDegraded(heartbeatState)
	isRecovered := heartbeatInfo.lastCheckState.IsRecovered(heartbeatState)
	allowNotify := now.Sub(heartbeatInfo.lastAlertTime) > m.cfg.NoticeInterval

	if isDegraded && allowNotify {
		return m.createErrorNotificationPackage(heartbeater, m.clock)
	} else if isRecovered {
		return m.createOkNotificationPackage(heartbeater, m.clock)
	}

	return nil
}

func (m *monitor) createErrorNotificationPackage(heartbeater heartbeat.Heartbeater, clock moira.Clock) *notifier.NotificationPackage {
	now := clock.NowUTC()

	m.heartbeatsInfo[heartbeater.Type()].lastAlertTime = now

	event := moira.NotificationEvent{
		Timestamp: now.Unix(),
		OldState:  moira.StateNODATA,
		State:     moira.StateERROR,
		Metric:    string(heartbeater.Type()),
		Value:     &errorValue,
	}

	trigger := moira.TriggerData{
		Name:       heartbeater.AlertSettings().Name,
		Desc:       heartbeater.AlertSettings().Desc,
		ErrorValue: triggerErrorValue,
	}

	return &notifier.NotificationPackage{
		Events:  []moira.NotificationEvent{event},
		Trigger: trigger,
	}
}

func (m *monitor) createOkNotificationPackage(heartbeater heartbeat.Heartbeater, clock moira.Clock) *notifier.NotificationPackage {
	now := clock.NowUTC()

	m.heartbeatsInfo[heartbeater.Type()].lastAlertTime = now

	event := moira.NotificationEvent{
		Timestamp: now.Unix(),
		OldState:  moira.StateERROR,
		State:     moira.StateOK,
		Metric:    string(heartbeater.Type()),
		Value:     &okValue,
	}

	trigger := moira.TriggerData{
		Name:       heartbeater.AlertSettings().Name,
		Desc:       heartbeater.AlertSettings().Desc,
		ErrorValue: triggerErrorValue,
	}

	return &notifier.NotificationPackage{
		Events:  []moira.NotificationEvent{event},
		Trigger: trigger,
	}
}

func (m *monitor) Stop() error {
	m.tomb.Kill(nil)
	return m.tomb.Wait()
}
