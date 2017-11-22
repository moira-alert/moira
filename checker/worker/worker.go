package worker

import (
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
)

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger   moira.Logger
	Database moira.Database
	Config   *checker.Config
	Metrics  *graphite.CheckerMetrics
	Cache    *cache.Cache
	lastData int64
	tomb     tomb.Tomb
}

// Start start schedule new MetricEvents and check for NODATA triggers
func (worker *Checker) Start() error {
	worker.lastData = time.Now().UTC().Unix()

	metricEventsChannel, err := worker.Database.SubscribeMetricEvents(&worker.tomb)
	if err != nil {
		return err
	}

	worker.tomb.Go(worker.noDataChecker)
	worker.Logger.Info("Moira Checker NoData checker started")

	worker.tomb.Go(func() error {
		return worker.metricsChecker(metricEventsChannel)
	})

	worker.Logger.Info("Moira Checker Checking new events started")
	return nil
}

// Stop stops checks triggers
func (worker *Checker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
