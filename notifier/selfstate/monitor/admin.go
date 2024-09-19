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
	adminCfg selfstate.AdminMonitorConfig
	database moira.Database
	notifier notifier.Notifier
}

func NewForAdmin(
	adminCfg selfstate.AdminMonitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
) (*monitor, error) {
	adminMonitor := adminMonitor{
		adminCfg: adminCfg,
		database: database,
		notifier: notifier,
	}

	cfg := monitorConfig{
		Name:           adminMonitorName,
		LockName:       adminMonitorLockName,
		LockTTL:        adminMonitorLockTTL,
		NoticeInterval: adminCfg.NoticeInterval,
		CheckInterval:  adminCfg.CheckInterval,
	}

	heartbeaters := createHearbeaters(adminCfg.HeartbeatsCfg, logger, database, clock)

	return newMonitor(
		cfg,
		logger,
		database,
		clock,
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
