package selfstate

import (
	"errors"
	"time"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	w "github.com/moira-alert/moira/worker"
)

var ErrDisabled = errors.New("moira Self State Monitoring disabled")
var ErrWrongConfig = errors.New("moira Self State Monitoring config is wrong")

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
	heartbeats []heartbeat.Heartbeater
}

func NewSelfCheckWorker(logger moira.Logger, database moira.Database, notifier notifier.Notifier, config Config) *SelfCheckWorker {
	heartbeats := createStandardHeartbeats(logger, database, config)
	return &SelfCheckWorker{Logger: logger, Database: database, Notifier: notifier, Config: config, heartbeats: heartbeats}
}

// Start self check worker
func (selfCheck *SelfCheckWorker) Start() error {
	if !selfCheck.Config.Enabled {
		return ErrDisabled
	}

	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		selfCheck.Logger.Errorf(ErrWrongConfig.Error())
		return ErrWrongConfig
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
		return ErrDisabled
	}
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		return ErrWrongConfig
	}

	selfCheck.tomb.Kill(nil)
	return selfCheck.tomb.Wait()
}

func createStandardHeartbeats(logger moira.Logger, database moira.Database, conf Config) []heartbeat.Heartbeater {
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

	if hb := heartbeat.GetNotifier(logger, database); hb != nil {
		heartbeats = append(heartbeats, hb)
	}

	return heartbeats
}
