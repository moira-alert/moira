package main

import (
	"fmt"
	"log"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/events"
	"github.com/moira-alert/moira-alert/notifier/notifications"
	"github.com/moira-alert/moira-alert/notifier/selfstate"
)

// NotifierService represents notifier functionality of moira
type NotifierService struct {
	Config          *notifier.Config
	SelfStateConfig *selfstate.Config
	DatabaseConfig  *redis.Config

	LogFile  string
	LogLevel string

	dataBase                 moira.Database
	selfState                *selfstate.SelfCheckWorker
	fetchEventsWorker        *events.FetchEventsWorker
	fetchNotificationsWorker *notifications.FetchNotificationsWorker
}

// Start Moira Notifier service
func (notifierService *NotifierService) Start() error {
	logger, err := logging.ConfigureLog(notifierService.LogFile, notifierService.LogLevel, "notifier")
	if err != nil {
		return fmt.Errorf("Can't configure logger for Notifier: %v", err)
	}

	notifierMetrics := metrics.ConfigureNotifierMetrics("notifier")

	notifierService.dataBase = redis.NewDatabase(logger, *notifierService.DatabaseConfig)

	sender := notifier.NewNotifier(notifierService.dataBase, logger, *notifierService.Config, notifierMetrics)
	if err = sender.RegisterSenders(notifierService.dataBase, notifierService.Config.FrontURL); err != nil {
		log.Fatalf("Can't configure senders: %s", err.Error())
	}

	notifierService.selfState = &selfstate.SelfCheckWorker{
		Log:      logger,
		DB:       notifierService.dataBase,
		Config:   *notifierService.SelfStateConfig,
		Notifier: sender,
	}
	if err = notifierService.selfState.Start(); err != nil {
		return fmt.Errorf("SelfState failed: %v", err)
	}
	notifierService.fetchEventsWorker = &events.FetchEventsWorker{
		Logger:    logger,
		Database:  notifierService.dataBase,
		Scheduler: notifier.NewScheduler(notifierService.dataBase, logger, notifierMetrics),
		Metrics:   notifierMetrics,
	}
	notifierService.fetchEventsWorker.Start()

	notifierService.fetchNotificationsWorker = &notifications.FetchNotificationsWorker{
		Logger:   logger,
		Database: notifierService.dataBase,
		Notifier: sender,
	}
	notifierService.fetchNotificationsWorker.Start()
	return nil
}

// Stop Moira Notifier service
func (notifierService *NotifierService) Stop() {
	notifierService.selfState.Stop()
	notifierService.fetchEventsWorker.Stop()
	notifierService.fetchNotificationsWorker.Stop()
	notifierService.dataBase.DeregisterBots()
}
