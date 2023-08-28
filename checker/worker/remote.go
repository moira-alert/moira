package worker

import (
	"time"

	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const (
	remoteTriggerLockName = "moira-remote-checker"
	remoteTriggerName     = "Remote checker"
)

type remoteChecker struct {
	check *Checker
}

// Returns the new instance of a CheckerWorker that checks graphite remote triggers
func NewRemoteChecker(check *Checker) CheckerWorker {
	return &remoteChecker{
		check: check,
	}
}

func (ch *remoteChecker) Name() string {
	return "Remote"
}

func (ch *remoteChecker) IsEnabled() bool {
	return ch.check.RemoteConfig.Enabled
}

func (ch *remoteChecker) MaxParallelChecks() int {
	return ch.check.Config.MaxParallelRemoteChecks
}

func (ch *remoteChecker) Metrics() *metrics.CheckMetrics {
	return ch.check.Metrics.RemoteMetrics
}

func (ch *remoteChecker) StartTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		ch.check.Logger,
		ch.check.Database.NewLock(remoteTriggerLockName, nodataCheckerLockTTL),
		ch.remoteTriggerChecker,
	).Run(ch.check.tomb.Dying())

	return nil
}

func (ch *remoteChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.check.Database.GetRemoteTriggersToCheck(count)
}

func (ch *remoteChecker) remoteTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.check.RemoteConfig.CheckInterval)
	ch.check.Logger.Info().Msg(remoteTriggerName + " started")
	for {
		select {
		case <-stop:
			ch.check.Logger.Info().Msg(remoteTriggerName + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := ch.checkRemote(); err != nil {
				ch.check.Logger.Error().
					Error(err).
					Msg("Remote trigger failed")
			}
		}
	}
}

func (ch *remoteChecker) checkRemote() error {
	source, err := ch.check.SourceProvider.GetRemote()
	if err != nil {
		return err
	}

	available, err := source.IsAvailable()
	if !available {
		ch.check.Logger.Info().
			Error(err).
			Msg("Remote API is unavailable. Stop checking remote triggers")
		return nil
	}

	ch.check.Logger.Debug().Msg("Checking remote triggers")

	triggerIds, err := ch.check.Database.GetRemoteTriggerIDs()
	if err != nil {
		return err
	}
	ch.addRemoteTriggerIDsIfNeeded(triggerIds)

	return nil
}

func (ch *remoteChecker) addRemoteTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckRemoteTriggerIDs := ch.check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckRemoteTriggerIDs) > 0 {
		ch.check.Database.AddRemoteTriggersToCheck(needToCheckRemoteTriggerIDs) //nolint
	}
}
