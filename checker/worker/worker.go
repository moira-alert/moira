package worker

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/metrics"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/remote"
	"github.com/moira-alert/moira/metric_source/vmselect"
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
	VMSelectConfig    *vmselect.Config
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

	err = check.startCheckerWorker(CheckerWorker{
		name:              "Local",
		enabled:           true,
		maxParallelChecks: check.Config.MaxParallelLocalChecks,
		metrics:           check.Metrics.LocalMetrics,

		triggerGetter:      check.localTriggerGetter,
		getTriggersToCheck: check.Database.GetLocalTriggersToCheck,
	})
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(CheckerWorker{
		name:              "Remote",
		enabled:           check.RemoteConfig.Enabled,
		maxParallelChecks: check.Config.MaxParallelRemoteChecks,
		metrics:           check.Metrics.RemoteMetrics,

		triggerGetter:      check.remoteTriggerGetter,
		getTriggersToCheck: check.Database.GetRemoteTriggersToCheck,
	})
	if err != nil {
		return err
	}

	err = check.startCheckerWorker(CheckerWorker{
		name:              "VMSelect",
		enabled:           check.VMSelectConfig.Enabled,
		maxParallelChecks: check.Config.MaxParallelVMSelectChecks,
		metrics:           check.Metrics.VMSelectMetrics,

		triggerGetter:      check.vmselectTriggerGetter,
		getTriggersToCheck: check.Database.GetVMSelectTriggersToCheck,
	})
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

type CheckerWorker struct {
	name               string
	enabled            bool
	maxParallelChecks  int
	triggerGetter      func() error
	getTriggersToCheck func(int) ([]string, error)
	metrics            *metrics.CheckMetrics
}

func (check *Checker) startCheckerWorker(w CheckerWorker) error {
	if !w.enabled {
		check.Logger.Info().Msg(w.name + " checker disabled")
		return nil
	}

	maxParallelChecks := w.maxParallelChecks
	if maxParallelChecks == 0 {
		maxParallelChecks = runtime.NumCPU()

		check.Logger.Info().
			Int("number_of_cpu", maxParallelChecks).
			Msg("MaxParallelRemoteChecks is not configured, set it to the number of CPU")
	}

	const maxParallelChecksMaxValue = 1024 * 8
	if maxParallelChecks > maxParallelChecksMaxValue {
		return errors.New("MaxParallel" + w.name + "Checks value is too large")
	}

	// ==== Go TriggerGetter ====
	check.tomb.Go(w.triggerGetter)
	check.Logger.Info().Msg(w.name + "checker started")

	// ==== Go startTriggerToCheckGetter ====
	triggerIdsToCheckChan := check.startTriggerToCheckGetter(
		w.getTriggersToCheck,
		maxParallelChecks,
	)

	for i := 0; i < maxParallelChecks; i++ {
		// ==== Go startTriggerHandler ====
		check.tomb.Go(func() error {
			return check.startTriggerHandler(
				triggerIdsToCheckChan,
				w.metrics,
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
