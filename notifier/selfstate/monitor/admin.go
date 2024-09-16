package monitor

import (
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate"
)

const (
	adminMonitorName     = "Moira Admin Selfstate Monitoring"
	adminMonitorLockName = "moira-admin-selfstate-monitor"
	adminMonitorLockTTL  = 15 * time.Second
)

type adminMonitor struct {
	adminCfg selfstate.AdminConfig
	database moira.Database
	notifier notifier.Notifier
}

func NewForAdmin(
	adminCfg selfstate.AdminConfig,
	logger moira.Logger,
	database moira.Database,
	notifier notifier.Notifier,
) (*monitor, error) {
	adminMonitor := adminMonitor{
		adminCfg: adminCfg,
		database: database,
		notifier: notifier,
	}

	cfg := monitorConfig{
		Name:           userMonitorName,
		LockName:       userMonitorLockName,
		LockTTL:        userMonitorLockTTL,
		NoticeInterval: adminCfg.NoticeInterval,
		CheckInterval:  adminCfg.CheckInterval,
	}

	heartbeaters := createHearbeaters(adminCfg.HeartbeatsCfg, logger, database)

	return newMonitor(
		cfg,
		logger,
		database,
		notifier,
		heartbeaters,
		adminMonitor.sendNotifications,
	)
}

func (am *adminMonitor) sendNotifications(pkgs []notifier.NotificationPackage) error {
	sendingWG := &sync.WaitGroup{}

	for _, pkg := range pkgs {
		for _, adminContact := range am.adminCfg.AdminContacts {
			contact := moira.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			}
			pkg.Contact = contact
			am.notifier.Send(&pkg, sendingWG)
			sendingWG.Wait()
		}
	}

	return nil
}
