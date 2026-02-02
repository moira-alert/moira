package selfstate

import (
	"time"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	w "github.com/moira-alert/moira/worker"
)

const (
	selfStateLockName = "moira-self-state-monitor"
	selfStateLockTTL  = time.Second * 15
)

// SelfCheckWorker checks what all notifier services works correctly and send message when moira don't work.
type SelfCheckWorker struct {
	Logger                  moira.Logger
	Database                moira.Database
	Notifier                notifier.Notifier
	Config                  Config
	oldState                moira.SelfStateWorkerState
	state                   moira.SelfStateWorkerState
	tomb                    tomb.Tomb
	heartbeatsGraph         heartbeatsGraph
	lastSuccessChecksResult graphExecutionResult
	lastChecksResult        graphExecutionResult
	clusterList             moira.ClusterList
	clock                   moira.Clock
}

// NewSelfCheckWorker creates SelfCheckWorker.
func NewSelfCheckWorker(logger moira.Logger, database moira.Database, notifier notifier.Notifier, config Config, clusterList moira.ClusterList, clock moira.Clock) *SelfCheckWorker {
	heartbeats := createStandardHeartbeats(logger, database, config, clusterList)

	return &SelfCheckWorker{
		Logger:          logger,
		Database:        database,
		Notifier:        notifier,
		Config:          config,
		heartbeatsGraph: heartbeats,
		clock:           clock,
	}
}

// Start self check worker.
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

// Stop self check worker and wait for finish.
func (selfCheck *SelfCheckWorker) Stop() error {
	selfCheck.tomb.Kill(nil)
	return selfCheck.tomb.Wait()
}

func createStandardHeartbeats(logger moira.Logger, database moira.Database, conf Config, clusterList moira.ClusterList) heartbeatsGraph {
	nowTS := time.Now().Unix()

	graph := heartbeatsGraph{
		[]heartbeat.Heartbeater{},
		[]heartbeat.Heartbeater{},
	}

	if hb := heartbeat.GetDatabase(conf.RedisDisconnectDelaySeconds, nowTS, conf.Checks.Database.SystemTags, logger, database); hb != nil {
		graph[0] = append(graph[0], hb)
	}

	if hb := heartbeat.GetFilter(conf.LastMetricReceivedDelaySeconds, nowTS, conf.Checks.Filter.SystemTags, logger, database); hb != nil {
		graph[1] = append(graph[1], hb)
	}

	if hb := heartbeat.GetLocalChecker(conf.LastCheckDelaySeconds, nowTS, conf.Checks.LocalChecker.SystemTags, logger, database); hb != nil && hb.NeedToCheckOthers() {
		graph[1] = append(graph[1], hb)
	}

	if hb := heartbeat.GetRemoteChecker(conf.LastRemoteCheckDelaySeconds, nowTS, conf.Checks.RemoteChecker.SystemTags, logger, database); hb != nil && hb.NeedToCheckOthers() {
		graph[1] = append(graph[1], hb)
	}

	for _, key := range clusterList {
		hb := heartbeat.GetNotifier(
			conf.Checks.Notifier.AnyClusterSourceTags,
			conf.Checks.Notifier.TagPrefixForClusterSource,
			conf.Checks.Notifier.LocalClusterSourceTags,
			key,
			logger,
			database,
		)
		if hb != nil {
			graph[1] = append(graph[1], hb)
		}
	}

	return graph
}
