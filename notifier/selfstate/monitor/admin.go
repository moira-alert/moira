package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
)

const (
	adminMonitorName     = "Moira Admin Selfstate Monitoring"
	adminMonitorLockName = "moira-admin-selfstate-monitor"
	adminMonitorLockTTL  = 15 * time.Second
)

type AdminMonitorConfig struct {
	MonitorBaseConfig

	AdminContacts []map[string]string `validate:"required_if=Enabled true"`
}

func (cfg AdminMonitorConfig) validate(senders map[string]bool) error {
	if err := moira.ValidateStruct(cfg); err != nil {
		return err
	}

	for _, contact := range cfg.AdminContacts {
		contactType := contact["type"]
		contactValue := contact["value"]

		if _, ok := senders[contactType]; !ok {
			return fmt.Errorf("unknown contact type in admin config: [%s]", contactType)
		}

		if contactValue == "" {
			return fmt.Errorf("value for [%s] must be present", contactType)
		}
	}

	return nil
}

type adminMonitor struct {
	adminCfg AdminMonitorConfig
	database moira.Database
	notifier notifier.Notifier
}

func NewForAdmin(
	adminCfg AdminMonitorConfig,
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
	notifier notifier.Notifier,
) (*monitor, error) {
	if err := adminCfg.validate(notifier.GetSenders()); err != nil {
		return nil, fmt.Errorf("admin config validation error: %w", err)
	}

	am := adminMonitor{
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

	heartbeaters := createHearbeaters(adminCfg.HeartbeatersCfg, logger, database, clock)

	return newMonitor(
		cfg,
		logger,
		database,
		clock,
		notifier,
		heartbeaters,
		am.sendNotifications,
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
