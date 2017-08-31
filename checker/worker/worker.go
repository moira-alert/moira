package worker

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"gopkg.in/tomb.v2"
	"time"
)

//Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger   moira.Logger
	Database moira.Database
	Config   *checker.Config
	Metrics  *graphite.CheckerMetrics
	lastData int64
	noCache  bool
	tomb     tomb.Tomb
}

//Start start schedule new MetricEvents and check for NODATA triggers
func (worker *Checker) Start() error {
	if !worker.Config.Enabled {
		worker.Logger.Debug("Checker Disabled")
		return nil
	}
	worker.lastData = time.Now().UTC().Unix()

	worker.tomb.Go(worker.noDataChecker)
	worker.Logger.Infof("NoData checker started")

	worker.tomb.Go(worker.metricsChecker)
	worker.Logger.Infof("Checking new events started")
	return nil
}

//Stop stops checks triggers
func (worker *Checker) Stop() error {
	if !worker.Config.Enabled {
		return nil
	}
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
