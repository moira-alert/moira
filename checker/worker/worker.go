package worker

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/metrics"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
)

// WorkerManager represents workers for periodically triggers checking based by new events
type WorkerManager struct {
	Logger   moira.Logger
	Database moira.Database

	Config         *checker.Config
	SourceProvider *metricSource.SourceProvider
	Metrics        *metrics.CheckerMetrics

	TriggerCache      *cache.Cache
	LazyTriggersCache *cache.Cache
	PatternCache      *cache.Cache
	lazyTriggerIDs    atomic.Value
	lastData          int64
	tomb              tomb.Tomb
}

// StartWorkers start schedule new MetricEvents and check for NODATA triggers
func (manager *WorkerManager) StartWorkers() error {
	var err error

	err = manager.startLazyTriggers()
	if err != nil {
		return err
	}

	err = manager.startLocalMetricEvents()
	if err != nil {
		return err
	}

	for clusterKey := range manager.Config.SourceCheckConfigs {
		validator, err := manager.makeSourceValidator(clusterKey)
		if err != nil {
			return err
		}

		checker, err := newUniversalChecker(manager, clusterKey, validator)
		if err != nil {
			return err
		}
		err = manager.startCheckerWorker(checker)
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *WorkerManager) makeSourceValidator(clusterKey moira.ClusterKey) (func() error, error) {
	if clusterKey.TriggerSource == moira.GraphiteLocal {
		return func() error {
			now := time.Now().UTC().Unix()

			if manager.lastData+manager.Config.StopCheckingIntervalSeconds < now {
				return nil
			}

			return fmt.Errorf("graphite local source invalid: no metrics for %d second", manager.Config.StopCheckingIntervalSeconds)
		}, nil
	}

	source, err := manager.SourceProvider.GetMetricSource(clusterKey)
	if err != nil {
		return nil, err
	}

	return func() error {
		if available, err := source.IsAvailable(); !available {
			return fmt.Errorf("source is not available: %w", err)
		}
		return nil
	}, nil
}

func (manager *WorkerManager) startLocalMetricEvents() error {
	if manager.Config.MetricEventPopBatchSize < 0 {
		return errors.New("MetricEventPopBatchSize param was less than zero")
	}

	if manager.Config.MetricEventPopBatchSize == 0 {
		manager.Config.MetricEventPopBatchSize = 100
	}

	subscribeMetricEventsParams := moira.SubscribeMetricEventsParams{
		BatchSize: manager.Config.MetricEventPopBatchSize,
		Delay:     manager.Config.MetricEventPopDelay,
	}

	metricEventsChannel, err := manager.Database.SubscribeMetricEvents(&manager.tomb, &subscribeMetricEventsParams)
	if err != nil {
		return err
	}

	localConfig, ok := manager.Config.SourceCheckConfigs[moira.MakeClusterKey(moira.GraphiteLocal, "default")]
	if !ok {
		return fmt.Errorf("can not initialize localMetricEvents: default local source is not configured")
	}

	for i := 0; i < localConfig.MaxParallelChecks; i++ {
		manager.tomb.Go(func() error {
			return manager.newMetricsHandler(metricEventsChannel)
		})
	}

	manager.tomb.Go(func() error {
		return manager.checkMetricEventsChannelLen(metricEventsChannel)
	})

	manager.Logger.Info().Msg("Checking new events started")

	go func() {
		<-manager.tomb.Dying()
		manager.Logger.Info().Msg("Checking for new events stopped")
	}()

	return nil
}

func (manager *WorkerManager) startCheckerWorker(w *universalChecker) error {
	const maxParallelChecksMaxValue = 1024 * 8
	if w.MaxParallelChecks() > maxParallelChecksMaxValue {
		return errors.New("MaxParallel" + w.Name() + "Checks value is too large")
	}

	manager.tomb.Go(w.StartTriggerScheduler)
	manager.Logger.Info().Msg(w.Name() + "checker started")

	triggerIdsToCheckChan := manager.startTriggerToCheckGetter(
		w.GetTriggersToCheck,
		w.MaxParallelChecks(),
	)

	for i := 0; i < w.MaxParallelChecks(); i++ {
		manager.tomb.Go(func() error {
			return manager.startTriggerHandler(
				triggerIdsToCheckChan,
				w.Metrics(),
			)
		})
	}

	return nil
}

func (manager *WorkerManager) startLazyTriggers() error {
	manager.lastData = time.Now().UTC().Unix()

	manager.lazyTriggerIDs.Store(make(map[string]bool))
	manager.tomb.Go(manager.lazyTriggersWorker)

	manager.tomb.Go(manager.checkTriggersToCheckCount)

	return nil
}

func (manager *WorkerManager) checkTriggersToCheckCount() error {
	/// TODO: Why we update metrics so frequently?
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-manager.tomb.Dying():
			return nil
		case <-checkTicker.C:
			for clusterKey := range manager.Config.SourceCheckConfigs {
				metrics, err := manager.Metrics.GetCheckMetricsBySource(clusterKey)
				if err != nil {
					/// TODO: log warn?
					continue
				}

				triggersToCheck, err := getTriggersToCheck(manager.Database, clusterKey)
				if err != nil {
					/// TODO: log warn?
					continue
				}
				metrics.TriggersToCheckCount.Update(triggersToCheck)
			}
		}
	}
}

func getTriggersToCheck(database moira.Database, clusterKey moira.ClusterKey) (int64, error) {
	return database.GetTriggersToCheckCount(clusterKey)
}

func (manager *WorkerManager) checkMetricEventsChannelLen(ch <-chan *moira.MetricEvent) error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-manager.tomb.Dying():
			return nil
		case <-checkTicker.C:
			manager.Metrics.MetricEventsChannelLen.Update(int64(len(ch)))
		}
	}
}

// Stop stops checks triggers
func (manager *WorkerManager) Stop() error {
	manager.tomb.Kill(nil)
	return manager.tomb.Wait()
}

func (manager *WorkerManager) addLocalTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckTriggerIDs := manager.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		manager.Database.AddLocalTriggersToCheck(needToCheckTriggerIDs) //nolint
	}
}
