package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

const (
	userMonitorName     = "Moira User Selfstate Monitoring"
	userMonitorLockName = "moira-user-selfstate-monitor"
	userMonitorLockTTL  = 15 * time.Second
)

type userMonitor struct {
	userCfg  selfstate.UserMonitorConfig
	database moira.Database
	notifier notifier.Notifier
}

func NewForUser(
	userCfg selfstate.UserMonitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
) (*monitor, error) {
	userMonitor := userMonitor{
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

	heartbeaters := createHearbeaters(userCfg.HeartbeatsCfg, logger, database, clock)

	return newMonitor(
		cfg,
		logger,
		database,
		clock,
		notifier,
		heartbeaters,
		userMonitor.sendNotifications,
	)
}

func (um *userMonitor) sendNotifications(pkgs []notifier.NotificationPackage) error {
	sendingWG := &sync.WaitGroup{}

	for _, pkg := range pkgs {
		event := pkg.Events[0]
		emergencyType := moira.EmergencyContactType(event.Metric)
		contactIDs, err := um.database.GetEmergencyTypeContactIDs(emergencyType)
		if err != nil {
			return fmt.Errorf("failed to get emergency type contact ids: %w", err)
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
