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

// Checker represents workers for periodically triggers checking based by new events
type Checker struct {
	Logger   moira.Logger
	Database moira.Database

	Config *checker.Config
	/// RemoteConfig      *remote.Config
	/// PrometheusConfig  *prometheus.Config

	SourceProvider *metricSource.SourceProvider
	Metrics        *metrics.CheckerMetrics

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

	err = check.startLazyTriggers()
	if err != nil {
		return err
	}

	err = check.startLocalMetricEvents()
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(newRemoteChecker(check, "default"))
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(newPrometheusChecker(check, "default"))
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(newLocalChecker(check, "default"))
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

	localConfig, ok := check.Config.SourceCheckConfigs[moira.MakeClusterKey(moira.GraphiteLocal, "default")]
	if !ok {
		return fmt.Errorf("can not initialize localMetricEvents: default local source is not configured")
	}

	for i := 0; i < localConfig.MaxParallelChecks; i++ {
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

type checkerWorker interface {
	// Returns the name of the worker for logging
	Name() string
	// Returns true if worker is enabled, false otherwise
	IsEnabled() bool
	// Returns the max number of parallel checks for this worker
	MaxParallelChecks() int
	// Returns the metrics for this worker
	Metrics() *metrics.CheckMetrics
	// Starts separate goroutine that fetches triggers for this worker from database and adds them to the check queue
	StartTriggerGetter() error
	// Fetches triggers from the queue
	GetTriggersToCheck(count int) ([]string, error)
}

// / Todo: remove ugly error passing
func (check *Checker) startCheckerWorker(w checkerWorker, err error) error {
	if err != nil {
		return err
	}

	if !w.IsEnabled() {
		check.Logger.Info().Msg(w.Name() + " checker disabled")
		return nil
	}

	const maxParallelChecksMaxValue = 1024 * 8
	if w.MaxParallelChecks() > maxParallelChecksMaxValue {
		return errors.New("MaxParallel" + w.Name() + "Checks value is too large")
	}

	check.tomb.Go(w.StartTriggerGetter)
	check.Logger.Info().Msg(w.Name() + "checker started")

	triggerIdsToCheckChan := check.startTriggerToCheckGetter(
		w.GetTriggersToCheck,
		w.MaxParallelChecks(),
	)

	for i := 0; i < w.MaxParallelChecks(); i++ {
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
	/// TODO: Why we update metrics so frequently?
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-check.tomb.Dying():
			return nil
		case <-checkTicker.C:
			for clusterKey, config := range check.Config.SourceCheckConfigs {
				if !config.Enabled {
					continue
				}

				metrics, err := check.Metrics.GetCheckMetricsBySource(clusterKey)
				if err != nil {
					/// TODO: log warn?
					continue
				}

				triggersToCheck, err := getTriggersToCheck(check.Database, clusterKey)
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
	switch clusterKey.TriggerSource {
	case moira.GraphiteLocal:
		return database.GetLocalTriggersToCheckCount()

	case moira.GraphiteRemote:
		return database.GetRemoteTriggersToCheckCount()

	case moira.PrometheusRemote:
		return database.GetPrometheusTriggersToCheckCount()

	default:
		return 0, fmt.Errorf("No triggers to check for cluster `%s`", clusterKey.String())
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
