package worker

import (
	"time"

	w "github.com/moira-alert/moira/worker"
)

const (
	vmselectTriggerLockName = "moira-vmselect-checker"
	vmselectTriggerName     = "VMSelect checker"
)

func (worker *Checker) vmselectTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		worker.Logger,
		worker.Database.NewLock(vmselectTriggerLockName, nodataCheckerLockTTL),
		worker.vmselectTriggerChecker,
	).Run(worker.tomb.Dying())

	return nil
}

func (worker *Checker) vmselectTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(worker.VMSelectConfig.CheckInterval)
	worker.Logger.Info().Msg(vmselectTriggerName + " started")
	for {
		select {
		case <-stop:
			worker.Logger.Info().Msg(vmselectTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := worker.checkVmselect(); err != nil {
				worker.Logger.Error().
					Error(err).
					// TODO: Remove this nonsence
					String("vmselect_trigger_name", remoteTriggerName).
					Msg("Vmselect trigger failed")
			}
		}
	}
}

func (worker *Checker) checkVmselect() error {
	// TODO: Generalise `IsAvailable`

	// source, err := worker.SourceProvider.GetVMSelect()
	// if err != nil {
	// 	return err
	// }
	// remoteAvailable, err := source.(*remote.Remote).IsRemoteAvailable()
	// if !remoteAvailable {
	// 	worker.Logger.Info().
	// 		Error(err).
	// 		Msg("Remote API is unavailable. Stop checking remote triggers")
	// } else {

	worker.Logger.Debug().Msg("Checking vmselect triggers")
	triggerIds, err := worker.Database.GetVMSelectTriggerIDs()
	if err != nil {
		return err
	}
	worker.addVMSelectTriggerIDsIfNeeded(triggerIds)

	// }

	return nil
}
