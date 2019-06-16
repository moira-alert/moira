package worker

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/remote"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
)

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger            moira.Logger
	Database          moira.Database
	Config            *checker.Config
	GraphiteConfig    *remote.Config
	SourceProvider    *metricSource.SourceProvider
	Metrics           *graphite.CheckerMetrics
	TriggerCache      *cache.Cache
	LazyTriggersCache *cache.Cache
	PatternCache      *cache.Cache
	lazyTriggerIDs    atomic.Value
	lastData          int64
	tomb              tomb.Tomb
	graphiteEnabled   bool
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

	worker.lazyTriggerIDs.Store(make(map[string]bool))
	worker.tomb.Go(worker.lazyTriggersWorker)

	worker.tomb.Go(worker.runNodataChecker)

	_, err = worker.SourceProvider.GetGraphite()
	worker.graphiteEnabled = err == nil

	if worker.graphiteEnabled && worker.Config.MaxParallelGraphiteChecks == 0 {
		worker.Config.MaxParallelGraphiteChecks = runtime.NumCPU()
		worker.Logger.Infof("MaxParallelGraphiteChecks is not configured, set it to the number of CPU - %d", worker.Config.MaxParallelGraphiteChecks)
	}

	if worker.graphiteEnabled {
		worker.tomb.Go(worker.graphiteChecker)
		worker.Logger.Info("Graphite checker started")
	} else {
		worker.Logger.Info("Graphite checker disabled")
	}

	worker.Logger.Infof("Start %v parallel local checker(s)", worker.Config.MaxParallelChecks)
	localTriggerIdsToCheckChan := worker.startTriggerToCheckGetter(worker.Database.GetLocalTriggersToCheck, worker.Config.MaxParallelChecks)
	for i := 0; i < worker.Config.MaxParallelChecks; i++ {
		worker.tomb.Go(func() error {
			return worker.newMetricsHandler(metricEventsChannel)
		})
		worker.tomb.Go(func() error {
			return worker.startTriggerHandler(localTriggerIdsToCheckChan, worker.Metrics.LocalMetrics)
		})
	}

	if worker.graphiteEnabled {
		worker.Logger.Infof("Start %v parallel graphite checker(s)", worker.Config.MaxParallelGraphiteChecks)
		graphiteTriggerIdsToCheckChan := worker.startTriggerToCheckGetter(worker.Database.GetGraphiteTriggersToCheck, worker.Config.MaxParallelGraphiteChecks)
		for i := 0; i < worker.Config.MaxParallelGraphiteChecks; i++ {
			worker.tomb.Go(func() error {
				return worker.startTriggerHandler(graphiteTriggerIdsToCheckChan, worker.Metrics.GraphiteMetrics)
			})
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
	var triggersToCheckCount, graphiteTriggersToCheckCount int64
	var err error
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		case <-checkTicker.C:
			triggersToCheckCount, err = worker.Database.GetLocalTriggersToCheckCount()
			if err == nil {
				worker.Metrics.LocalMetrics.TriggersToCheckCount.Update(triggersToCheckCount)
			}
			if worker.graphiteEnabled {
				graphiteTriggersToCheckCount, err = worker.Database.GetGraphiteTriggersToCheckCount()
				if err == nil {
					worker.Metrics.GraphiteMetrics.TriggersToCheckCount.Update(graphiteTriggersToCheckCount)
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
