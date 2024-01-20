package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const (
	prometheusTriggerLockName = "moira-prometheus-checker"
	prometheusTriggerName     = "Prometheus checker"
)

type prometheusChecker struct {
	metrics           *metrics.CheckMetrics
	sourceCheckConfig checker.SourceCheckConfig
	source            metricSource.MetricSource
	check             *Checker
}

func newPrometheusChecker(check *Checker, clusterId string) (checkerWorker, error) {
	key := moira.MakeClusterKey(moira.GraphiteRemote, clusterId)

	metrics, err := check.Metrics.GetCheckMetricsBySource(key)
	if err != nil {
		return nil, err
	}

	source, err := check.SourceProvider.GetMetricSource(key)
	if err != nil {
		return nil, err
	}

	return &prometheusChecker{
		check:             check,
		sourceCheckConfig: check.Config.SourceCheckConfigs[key],
		source:            source,
		metrics:           metrics,
	}, nil
}

func (ch *prometheusChecker) Name() string {
	return "Prometheus"
}

func (ch *prometheusChecker) IsEnabled() bool {
	return ch.sourceCheckConfig.Enabled
}

func (ch *prometheusChecker) MaxParallelChecks() int {
	return ch.sourceCheckConfig.MaxParallelChecks
}

func (ch *prometheusChecker) Metrics() *metrics.CheckMetrics {
	return ch.metrics
}

func (ch *prometheusChecker) StartTriggerGetter() error {
	w.NewWorker(
		remoteTriggerName,
		ch.check.Logger,
		ch.check.Database.NewLock(prometheusTriggerLockName, nodataCheckerLockTTL),
		ch.prometheusTriggerChecker,
	).Run(ch.check.tomb.Dying())

	return nil
}

func (ch *prometheusChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.check.Database.GetPrometheusTriggersToCheck(count)
}

func (ch *prometheusChecker) prometheusTriggerChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.sourceCheckConfig.CheckInterval)
	ch.check.Logger.Info().Msg(prometheusTriggerName + " started")
	for {
		select {
		case <-stop:
			ch.check.Logger.Info().Msg(prometheusTriggerName + " stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := ch.checkPrometheus(); err != nil {
				ch.check.Logger.Error().
					Error(err).
					Msg("Prometheus trigger failed")
			}
		}
	}
}

func (ch *prometheusChecker) checkPrometheus() error {
	source := ch.source

	available, err := source.IsAvailable()
	if !available {
		ch.check.Logger.Info().
			Error(err).
			Msg("Prometheus API is unavailable. Stop checking prometheus triggers")
		return nil
	}

	ch.check.Logger.Debug().Msg("Checking prometheus triggers")
	triggerIds, err := ch.check.Database.GetPrometheusTriggerIDs()

	if err != nil {
		return err
	}

	ch.addPrometheusTriggerIDsIfNeeded(triggerIds)

	return nil
}

func (ch *prometheusChecker) addPrometheusTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckPrometheusTriggerIDs := ch.check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckPrometheusTriggerIDs) > 0 {
		ch.check.Database.AddPrometheusTriggersToCheck(needToCheckPrometheusTriggerIDs) //nolint
	}
}
