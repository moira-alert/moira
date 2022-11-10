package worker

import (
	"time"

	"github.com/moira-alert/moira"
	w "github.com/moira-alert/moira/worker"
)

const (
	workerName     = "METRIC EVENT HANDLER"
	workerLockName = "metric-event-handler-lock"
	workerLockTTL  = time.Second * 15
)

// Method handles metric events that come from filter through pubsub channel and
// converts them into tiggersIds that should be updated.
func (worker *Checker) handleMetricEvents() error {
	w.NewWorker(
		workerName,
		worker.Logger,
		worker.Database.NewLock(workerLockName, workerLockTTL),
		worker.startHandleMetricEvents,
	).Run(worker.tomb.Dying())

	return nil
}

func (worker *Checker) startHandleMetricEvents(stop <-chan struct{}) error {
	metricEventsChannel, err := worker.Database.SubscribeMetricEvents(stop)

	if err != nil {
		return err
	}

	for i := 0; i < worker.Config.MaxParallelChecks; i++ {
		go func() error {
			return worker.newMetricsHandler(metricEventsChannel, stop)
		}()
	}

	worker.tomb.Go(func() error { return worker.checkMetricEventsChannelLen(metricEventsChannel, stop) })

	<-stop
	return nil
}

func (worker *Checker) checkMetricEventsChannelLen(ch <-chan *moira.MetricEvent, stop <-chan struct{}) error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-stop:
			return nil
		case <-checkTicker.C:
			worker.Metrics.MetricEventsChannelLen.Update(int64(len(ch)))
		}
	}
}
