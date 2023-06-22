package worker

import (
	"time"

	w "github.com/moira-alert/moira/worker"
)

const (
	vmselectTriggerLockName = "moira-vmselect-checker"
	vmselectTriggerName     = "VMSelect checker"
)

func (check *Checker) vmselectTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		check.Logger,
		check.Database.NewLock(vmselectTriggerLockName, nodataCheckerLockTTL),
		check.vmselectTriggerChecker,
	).Run(check.tomb.Dying())

	return nil
}

func (check *Checker) vmselectTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(check.VMSelectConfig.CheckInterval)
	check.Logger.Info().Msg(vmselectTriggerName + " started")
	for {
		select {
		case <-stop:
			check.Logger.Info().Msg(vmselectTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := check.checkVmselect(); err != nil {
				check.Logger.Error().
					Error(err).
					Msg("Vmselect trigger failed")
			}
		}
	}
}

func (check *Checker) checkVmselect() error {
	source, err := check.SourceProvider.GetVMSelect()
	if err != nil {
		return err
	}

	available, err := source.IsAvailable()
	if !available {
		check.Logger.Info().
			Error(err).
			Msg("VMSelect API is unavailable. Stop checking vmselect triggers")
		return nil
	}

	check.Logger.Debug().Msg("Checking vmselect triggers")
	triggerIds, err := check.Database.GetVMSelectTriggerIDs()

	if err != nil {
		return err
	}

	check.addVMSelectTriggerIDsIfNeeded(triggerIds)

	return nil
}
