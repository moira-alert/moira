package worker

import (
	"time"

	w "github.com/moira-alert/moira/worker"
)

const (
	remoteTriggerLockName = "moira-remote-checker"
	remoteTriggerName     = "Remote checker"
)

func (check *Checker) remoteTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		check.Logger,
		check.Database.NewLock(remoteTriggerLockName, nodataCheckerLockTTL),
		check.remoteTriggerChecker,
	).Run(check.tomb.Dying())

	return nil
}

func (check *Checker) remoteTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(check.RemoteConfig.CheckInterval)
	check.Logger.Info().Msg(remoteTriggerName + " started")
	for {
		select {
		case <-stop:
			check.Logger.Info().Msg(remoteTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := check.checkRemote(); err != nil {
				check.Logger.Error().
					Error(err).
					Msg("Remote trigger failed")
			}
		}
	}
}

func (check *Checker) checkRemote() error {
	source, err := check.SourceProvider.GetRemote()
	if err != nil {
		return err
	}

	available, err := source.IsAvailable()
	if !available {
		check.Logger.Info().
			Error(err).
			Msg("Remote API is unavailable. Stop checking remote triggers")
		return nil
	}

	check.Logger.Debug().Msg("Checking remote triggers")

	triggerIds, err := check.Database.GetRemoteTriggerIDs()
	if err != nil {
		return err
	}
	check.addRemoteTriggerIDsIfNeeded(triggerIds)

	return nil
}
