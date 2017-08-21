package worker

import (
	"github.com/moira-alert/moira-alert"
	"gopkg.in/tomb.v2"
	"time"
)

func (worker *Worker) metricsChecker() error {
	metricEventsChannel := worker.Database.SubscribeMetricEvents(&worker.tomb)
	var handleTomb tomb.Tomb
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			handleTomb.Wait()
			worker.Logger.Info("Checking for new event stopped")
			return nil
		}
		if !handleTomb.Alive() {
			handleTomb = tomb.Tomb{}
		}
		handleTomb.Go(func() error { return worker.handle(metricEvent) })
	}
	return nil
}

func (worker *Worker) handle(metricEvent *moira.MetricEvent) error {
	if err := worker.handleMetricEvent(metricEvent); err != nil {
		worker.Logger.Errorf("Failed to handle metricEvent", err.Error())
	}
	return nil
}

func (worker *Worker) handleMetricEvent(metricEvent *moira.MetricEvent) error {
	worker.lastData = time.Now().UTC().Unix()
	pattern := metricEvent.Pattern
	metric := metricEvent.Metric

	if err := worker.Database.AddPatternMetric(pattern, metric); err != nil {
		return err
	}
	triggerIds, err := worker.Database.GetPatternTriggerIds(pattern)
	if err != nil {
		return err
	}
	if len(triggerIds) == 0 {
		if err := worker.Database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}
	var performTomb tomb.Tomb
	worker.perform(triggerIds, worker.noCache, worker.Config.CheckInterval, &performTomb)
	performTomb.Wait()
	return nil
}
