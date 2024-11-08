package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	"github.com/moira-alert/moira/notifier"
)

const (
	userMonitorName     = "Moira User Selfstate Monitoring"
	userMonitorLockName = "moira-user-selfstate-monitor"
	userMonitorLockTTL  = 15 * time.Second
)

// UserMonitorConfig defines the configuration of the user monitor.
type UserMonitorConfig struct {
	MonitorBaseConfig
}

type userMonitor struct {
	userCfg  UserMonitorConfig
	database moira.Database
	notifier notifier.Notifier
}

// NewForUser is a method to create a user monitor.
func NewForUser(
	userCfg UserMonitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
) (*monitor, error) {
	if err := moira.ValidateStruct(userCfg); err != nil {
		return nil, fmt.Errorf("user config validation error: %w", err)
	}

	um := userMonitor{
		userCfg:  userCfg,
		database: database,
		notifier: notifier,
	}

	cfg := monitorConfig{
		Name:           userMonitorName,
		LockName:       userMonitorLockName,
		LockTTL:        userMonitorLockTTL,
		NoticeInterval: userCfg.NoticeInterval,
		CheckInterval:  userCfg.CheckInterval,
	}

	heartbeaters := createHearbeaters(userCfg.HeartbeatersCfg, logger, database, clock)

	return newMonitor(
		cfg,
		logger,
		database,
		clock,
		notifier,
		heartbeaters,
		um.sendNotifications,
	)
}

func (um *userMonitor) sendNotifications(pkgs []notifier.NotificationPackage) error {
	sendingWG := &sync.WaitGroup{}

	for _, pkg := range pkgs {
		event := pkg.Events[0]
		heartbeatType := datatypes.HeartbeatType(event.Metric)
		contactIDs, err := um.database.GetHeartbeatTypeContactIDs(heartbeatType)
		if err != nil {
			return fmt.Errorf("failed to get heartbeat type contact ids: %w", err)
		}

		contacts, err := um.database.GetContacts(contactIDs)
		if err != nil {
			return fmt.Errorf("failed to get contacts by ids: %w", err)
		}

		for _, contact := range contacts {
			if contact != nil {
				pkg.Contact = *contact
				um.notifier.Send(&pkg, sendingWG)
				sendingWG.Wait()
			}
		}
	}

	return nil
}
