package worker

import (
	"time"

	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const (
	vmselectTriggerLockName = "moira-vmselect-checker"
	vmselectTriggerName     = "VMSelect checker"
)

type vmselectChecker struct {
	check *Checker
}

func NewVMSelectChecker(check *Checker) CheckerWorker {
	return &vmselectChecker{
		check: check,
	}
}

func (ch *vmselectChecker) Name() string {
	return "VMSelect"
}

func (ch *vmselectChecker) IsEnabled() bool {
	return ch.check.VMSelectConfig.Enabled
}

func (ch *vmselectChecker) MaxParallelChecks() int {
	return ch.check.Config.MaxParallelVMSelectChecks
}

func (ch *vmselectChecker) Metrics() *metrics.CheckMetrics {
	return ch.check.Metrics.VMSelectMetrics
}

func (ch *vmselectChecker) StartTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		ch.check.Logger,
		ch.check.Database.NewLock(vmselectTriggerLockName, nodataCheckerLockTTL),
		ch.vmselectTriggerChecker,
	).Run(ch.check.tomb.Dying())

	return nil
}

func (ch *vmselectChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.check.Database.GetVMSelectTriggersToCheck(count)
}

func (ch *vmselectChecker) vmselectTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.check.VMSelectConfig.CheckInterval)
	ch.check.Logger.Info().Msg(vmselectTriggerName + " started")
	for {
		select {
		case <-stop:
			ch.check.Logger.Info().Msg(vmselectTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := ch.checkVmselect(); err != nil {
				ch.check.Logger.Error().
					Error(err).
					Msg("Vmselect trigger failed")
			}
		}
	}
}

func (ch *vmselectChecker) checkVmselect() error {
	source, err := ch.check.SourceProvider.GetVMSelect()
	if err != nil {
		return err
	}

	available, err := source.IsAvailable()
	if !available {
		ch.check.Logger.Info().
			Error(err).
			Msg("VMSelect API is unavailable. Stop checking vmselect triggers")
		return nil
	}

	ch.check.Logger.Debug().Msg("Checking vmselect triggers")
	triggerIds, err := ch.check.Database.GetVMSelectTriggerIDs()

	if err != nil {
		return err
	}

	ch.addVMSelectTriggerIDsIfNeeded(triggerIds)

	return nil
}

func (ch *vmselectChecker) addVMSelectTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckVMSelectTriggerIDs := ch.check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckVMSelectTriggerIDs) > 0 {
		ch.check.Database.AddVMSelectTriggersToCheck(needToCheckVMSelectTriggerIDs) //nolint
	}
}
