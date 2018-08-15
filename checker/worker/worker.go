package worker

import (
	"runtime"
	"time"

	"github.com/moira-alert/moira/remote"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
)

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger        moira.Logger
	Database      moira.Database
	Config        *checker.Config
	RemoteConfig  *remote.Config
	Metrics       *graphite.CheckerMetrics
	TriggerCache  *cache.Cache
	PatternCache  *cache.Cache
	lastData      int64
	tomb          tomb.Tomb
	remoteEnabled bool
}

// Start start schedule new MetricEvents and check for NODATA triggers
func (worker *Checker) Start() error {
	if worker.Config.MaxParallelChecks == 0 {
		worker.Config.MaxParallelChecks = runtime.NumCPU()
		worker.Logger.Infof("MaxParallelChecks is not configured, set it to the number of CPU - %d", worker.Config.MaxParallelChecks)
	}

	worker.lastData = time.Now().UTC().Unix()

	metricEventsChannel, err := worker.Database.SubscribeMetricEvents(&worker.tomb)
	if err != nil {
		return err
	}

	worker.tomb.Go(worker.noDataChecker)
	worker.Logger.Info("NODATA checker started")

	worker.remoteEnabled = worker.RemoteConfig.IsEnabled()

	if worker.remoteEnabled && worker.Config.MaxParallelRemoteChecks == 0 {
		worker.Config.MaxParallelRemoteChecks = runtime.NumCPU()
		worker.Logger.Infof("MaxParallelRemoteChecks is not configured, set it to the number of CPU - %d", worker.Config.MaxParallelRemoteChecks)
	}

	if worker.remoteEnabled {
		worker.tomb.Go(worker.remoteChecker)
		worker.Logger.Info("Remote checker started")
	} else {
		worker.Logger.Info("Remote checker disabled")
	}

	worker.Logger.Infof("Start %v parallel checker(s)", worker.Config.MaxParallelChecks)
	for i := 0; i < worker.Config.MaxParallelChecks; i++ {
		worker.tomb.Go(func() error { return worker.metricsChecker(metricEventsChannel) })
		worker.tomb.Go(func() error { return worker.startTriggerHandler(false, worker.Metrics.MoiraMetrics) })
	}

	if worker.remoteEnabled {
		worker.Logger.Infof("Start %v parallel remote checker(s)", worker.Config.MaxParallelRemoteChecks)
		for i := 0; i < worker.Config.MaxParallelRemoteChecks; i++ {
			worker.tomb.Go(func() error { return worker.startTriggerHandler(true, worker.Metrics.RemoteMetrics) })
		}
	}
	worker.Logger.Info("Checking new events started")

	go func() {
		<-worker.tomb.Dying()
		worker.Logger.Info("Checking for new events stopped")
	}()

	worker.tomb.Go(func() error { return worker.checkMetricEventsChannelLen(metricEventsChannel) })
	worker.tomb.Go(worker.checkTriggersToCheckCount)
	return nil
}

func (worker *Checker) checkTriggersToCheckCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	var triggersToCheckCount, remoteTriggersToCheckCount int64
	var err error
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		case <-checkTicker.C:
			triggersToCheckCount, err = worker.Database.GetTriggersToCheckCount()
			if err == nil {
				worker.Metrics.MoiraMetrics.TriggersToCheckCount.Update(triggersToCheckCount)
			}
			if worker.remoteEnabled {
				remoteTriggersToCheckCount, err = worker.Database.GetRemoteTriggersToCheckCount()
				if err == nil {
					worker.Metrics.RemoteMetrics.TriggersToCheckCount.Update(remoteTriggersToCheckCount)
				}
			}
		}
	}
}

func (worker *Checker) checkMetricEventsChannelLen(ch <-chan *moira.MetricEvent) error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		case <-checkTicker.C:
			worker.Metrics.MetricEventsChannelLen.Update(int64(len(ch)))
		}
	}
}

// Stop stops checks triggers
func (worker *Checker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
