package worker

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/metrics"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/prometheus"
	"github.com/moira-alert/moira/metric_source/remote"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
)

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger            moira.Logger
	Database          moira.Database
	Config            *checker.Config
	RemoteConfig      *remote.Config
	PrometheusConfig  *prometheus.Config
	SourceProvider    *metricSource.SourceProvider
	Metrics           *metrics.CheckerMetrics
	TriggerCache      *cache.Cache
	LazyTriggersCache *cache.Cache
	PatternCache      *cache.Cache
	lazyTriggerIDs    atomic.Value
	lastData          int64
	tomb              tomb.Tomb
}

// Start start schedule new MetricEvents and check for NODATA triggers
func (check *Checker) Start() error {
	var err error

	err = check.startLocalMetricEvents()
	if err != nil {
		return err
	}

	err = check.startLazyTriggers()
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(NewRemoteChecker(check))
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(NewPrometheusChecker(check))
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(NewLocalChecker(check))
	if err != nil {
		return err
	}

	return nil
}

func (check *Checker) startLocalMetricEvents() error {
	if check.Config.MetricEventPopBatchSize < 0 {
		return errors.New("MetricEventPopBatchSize param was less than zero")
	}

	if check.Config.MetricEventPopBatchSize == 0 {
		check.Config.MetricEventPopBatchSize = 100
	}

	subscribeMetricEventsParams := moira.SubscribeMetricEventsParams{
		BatchSize: check.Config.MetricEventPopBatchSize,
		Delay:     check.Config.MetricEventPopDelay,
	}

	metricEventsChannel, err := check.Database.SubscribeMetricEvents(&check.tomb, &subscribeMetricEventsParams)
	if err != nil {
		return err
	}

	for i := 0; i < check.Config.MaxParallelLocalChecks; i++ {
		check.tomb.Go(func() error {
			return check.newMetricsHandler(metricEventsChannel)
		})
	}

	check.tomb.Go(func() error {
		return check.checkMetricEventsChannelLen(metricEventsChannel)
	})

	check.Logger.Info().Msg("Checking new events started")

	go func() {
		<-check.tomb.Dying()
		check.Logger.Info().Msg("Checking for new events stopped")
	}()

	return nil
}

// Interface that represents the worker that checks triggers from the specific metric source
type CheckerWorker interface {
	Name() string
	IsEnabled() bool
	MaxParallelChecks() int
	Metrics() *metrics.CheckMetrics

	StartTriggerGetter() error
	GetTriggersToCheck(count int) ([]string, error)
}

func (check *Checker) startCheckerWorker(w CheckerWorker) error {
	if !w.IsEnabled() {
		check.Logger.Info().Msg(w.Name() + " checker disabled")
		return nil
	}

	maxParallelChecks := w.MaxParallelChecks()
	if maxParallelChecks == 0 {
		maxParallelChecks = runtime.NumCPU()

		check.Logger.Info().
			Int("number_of_cpu", maxParallelChecks).
			Msg("MaxParallel" + w.Name() + "Checks is not configured, set it to the number of CPU")
	}

	const maxParallelChecksMaxValue = 1024 * 8
	if maxParallelChecks > maxParallelChecksMaxValue {
		return errors.New("MaxParallel" + w.Name() + "Checks value is too large")
	}

	check.tomb.Go(w.StartTriggerGetter)
	check.Logger.Info().Msg(w.Name() + "checker started")

	triggerIdsToCheckChan := check.startTriggerToCheckGetter(
		w.GetTriggersToCheck,
		maxParallelChecks,
	)

	for i := 0; i < maxParallelChecks; i++ {
		check.tomb.Go(func() error {
			return check.startTriggerHandler(
				triggerIdsToCheckChan,
				w.Metrics(),
			)
		})
	}

	return nil
}

func (check *Checker) startLazyTriggers() error {
	check.lastData = time.Now().UTC().Unix()

	check.lazyTriggerIDs.Store(make(map[string]bool))
	check.tomb.Go(check.lazyTriggersWorker)

	check.tomb.Go(check.checkTriggersToCheckCount)

	return nil
}

func (check *Checker) checkTriggersToCheckCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	var triggersToCheckCount, remoteTriggersToCheckCount int64
	var err error
	for {
		select {
		case <-check.tomb.Dying():
			return nil
		case <-checkTicker.C:
			triggersToCheckCount, err = check.Database.GetLocalTriggersToCheckCount()
			if err == nil {
				check.Metrics.LocalMetrics.TriggersToCheckCount.Update(triggersToCheckCount)
			}
			if check.RemoteConfig.Enabled {
				remoteTriggersToCheckCount, err = check.Database.GetRemoteTriggersToCheckCount()
				if err == nil {
					check.Metrics.RemoteMetrics.TriggersToCheckCount.Update(remoteTriggersToCheckCount)
				}
			}
		}
	}
}

func (check *Checker) checkMetricEventsChannelLen(ch <-chan *moira.MetricEvent) error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-check.tomb.Dying():
			return nil
		case <-checkTicker.C:
			check.Metrics.MetricEventsChannelLen.Update(int64(len(ch)))
		}
	}
}

// Stop stops checks triggers
func (check *Checker) Stop() error {
	check.tomb.Kill(nil)
	return check.tomb.Wait()
}
