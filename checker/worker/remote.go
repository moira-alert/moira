package worker

import (
	"time"

	"github.com/moira-alert/moira/metric_source/remote"
	w "github.com/moira-alert/moira/worker"
)

const (
	remoteTriggerLockName = "moira-remote-checker"
	remoteTriggerName     = "Remote checker"
)

func (worker *Checker) remoteTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		worker.Logger,
		worker.Database.NewLock(remoteTriggerLockName, nodataCheckerLockTTL),
		worker.remoteTriggerChecker,
	).Run(worker.tomb.Dying())

	return nil
}

func (worker *Checker) remoteTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(worker.RemoteConfig.CheckInterval)
	worker.Logger.Info().Msg(remoteTriggerName + " started")
	for {
		select {
		case <-stop:
			worker.Logger.Info().Msg(remoteTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := worker.checkRemote(); err != nil {
				worker.Logger.Error().
					Error(err).
					String("remote_trigger_name", remoteTriggerName).
					Msg("Remote trigger failed")
			}
		}
	}
}

func (worker *Checker) checkRemote() error {
	source, err := worker.SourceProvider.GetRemote()
	if err != nil {
		return err
	}
	remoteAvailable, err := source.(*remote.Remote).IsRemoteAvailable()
	if !remoteAvailable {
		worker.Logger.Info().
			Error(err).
			Msg("Remote API is unavailable. Stop checking remote triggers")
	} else {
		worker.Logger.Debug().Msg("Checking remote triggers")
		triggerIds, err := worker.Database.GetRemoteTriggerIDs()
		if err != nil {
			return err
		}
		worker.addRemoteTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
