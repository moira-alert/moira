package selfstate

import (
	"time"

	"github.com/moira-alert/moira/metrics"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	w "github.com/moira-alert/moira/worker"
)

const selfStateLockName = "moira-self-state-monitor"
const selfStateLockTTL = time.Second * 15

// SelfCheckWorker checks what all notifier services works correctly and send message when moira don't work
type SelfCheckWorker struct {
	Logger     moira.Logger
	Database   moira.Database
	Notifier   notifier.Notifier
	Config     Config
	tomb       tomb.Tomb
	heartbeats []heartbeat.Heartbeater
}

// NewSelfCheckWorker creates SelfCheckWorker.
func NewSelfCheckWorker(logger moira.Logger, database moira.Database, notifier notifier.Notifier, config Config, metrics *metrics.HeartBeatMetrics) *SelfCheckWorker {
	heartbeats := createStandardHeartbeats(logger, database, config, metrics)
	return &SelfCheckWorker{Logger: logger, Database: database, Notifier: notifier, Config: config, heartbeats: heartbeats}
}

// Start self check worker
func (selfCheck *SelfCheckWorker) Start() error {
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		return err
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
	selfCheck.tomb.Kill(nil)
	return selfCheck.tomb.Wait()
}

func createStandardHeartbeats(logger moira.Logger, database moira.Database, conf Config, metrics *metrics.HeartBeatMetrics) []heartbeat.Heartbeater {
	heartbeats := make([]heartbeat.Heartbeater, 0)

	if hb := heartbeat.GetDatabase(conf.RedisDisconnectDelaySeconds, logger, database); hb != nil {
		heartbeats = append(heartbeats, hb)
	}

	if hb := heartbeat.GetFilter(conf.LastMetricReceivedDelaySeconds, logger, database); hb != nil {
		heartbeats = append(heartbeats, hb)
	}

	if hb := heartbeat.GetLocalChecker(conf.LastCheckDelaySeconds, logger, database); hb != nil && hb.NeedToCheckOthers() {
		heartbeats = append(heartbeats, hb)
	}

	if hb := heartbeat.GetRemoteChecker(conf.LastRemoteCheckDelaySeconds, logger, database); hb != nil && hb.NeedToCheckOthers() {
		heartbeats = append(heartbeats, hb)
	}

	if hb := heartbeat.GetNotifier(logger, database, metrics); hb != nil {
		heartbeats = append(heartbeats, hb)
	}

	return heartbeats
}
