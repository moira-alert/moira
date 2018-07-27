package worker

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
)

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger          moira.Logger
	Database        moira.Database
	Config          *checker.Config
	Metrics         *graphite.CheckerMetrics
	TriggerCache    *cache.Cache
	PatternCache    *cache.Cache
	lastData        int64
	tomb            tomb.Tomb
	triggersToCheck chan string
}

// Start start schedule new MetricEvents and check for NODATA triggers
func (worker *Checker) Start() error {
	if worker.Config.MaxParallelChecks == 0 {
		return fmt.Errorf("MaxParallelChecks is not configured, checker does not start")
	}

	worker.lastData = time.Now().UTC().Unix()
	worker.triggersToCheck = make(chan string, 16384)

	metricEventsChannel, err := worker.Database.SubscribeMetricEvents(&worker.tomb)
	if err != nil {
		return err
	}

	worker.tomb.Go(worker.noDataChecker)
	worker.Logger.Info("NODATA checker started")

	worker.Logger.Infof("Start %v parallel checkers", worker.Config.MaxParallelChecks)
	for i := 0; i < worker.Config.MaxParallelChecks; i++ {
		worker.tomb.Go(func() error { return worker.metricsChecker(metricEventsChannel) })
		worker.tomb.Go(worker.startTriggerHandler)
	}
	worker.Logger.Info("Checking new events started")

	go func() {
		for {
			<-worker.tomb.Dying()
			close(worker.triggersToCheck)
			worker.Logger.Info("Checking for new events stopped")
			return
		}
	}()

	worker.tomb.Go(worker.checkTriggersToCheckChannelLen)
	worker.tomb.Go(func() error { return worker.checkMetricEventsChannelLen(metricEventsChannel) })
	return nil
}

func (worker *Checker) checkTriggersToCheckChannelLen() error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		case <-checkTicker.C:
			worker.Metrics.TriggersToCheckChannelLen.Update(int64(len(worker.triggersToCheck)))
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
