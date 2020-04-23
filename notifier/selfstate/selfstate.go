package selfstate

import (
	"time"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	w "github.com/moira-alert/moira/worker"
)

var defaultCheckInterval = time.Second * 10

const selfStateLockName = "moira-self-state-monitor"
const selfStateLockTTL = time.Second * 15

// SelfCheckWorker checks what all notifier services works correctly and send message when moira don't work
type SelfCheckWorker struct {
	Logger     moira.Logger
	Database   moira.Database
	Notifier   notifier.Notifier
	Config     Config
	tomb       tomb.Tomb
	Heartbeats []heartbeat.Heartbeater
}

// Start self check worker
func (selfCheck *SelfCheckWorker) Start() error {
	if !selfCheck.Config.Enabled {
		selfCheck.Logger.Debugf("Moira Self State Monitoring disabled")
		return nil
	}
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		selfCheck.Logger.Errorf("Can't configure Moira Self State Monitoring: %s", err.Error())
		return nil
	}

	selfCheck.tomb.Go(func() error {
		w.NewWorker(
			"Moira Self State Monitoring",
			selfCheck.Logger,
			selfCheck.Database.NewLock(selfStateLockName, selfStateLockTTL),
			selfCheck.selfStateChecker,
		).Run(selfCheck.tomb.Dying())
		return nil
	})

	return nil
}

// Stop self check worker and wait for finish
func (selfCheck *SelfCheckWorker) Stop() error {
	if !selfCheck.Config.Enabled {
		return nil
	}
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		return nil
	}

	selfCheck.tomb.Kill(nil)
	return selfCheck.tomb.Wait()
}
