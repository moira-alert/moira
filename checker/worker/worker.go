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

	remoteEnabled   bool
	vmselectEnabled bool
}

// TODO: Refactor this function, it's literally unreadable
// Start start schedule new MetricEvents and check for NODATA triggers
func (worker *Checker) Start() error {
	var err error

	err = worker.startLocal()
	if err != nil {
		return err
	}

	err = worker.startRemote()
	if err != nil {
		return err
	}

	err = worker.startVMSelect()
	if err != nil {
		return err
	}

	return nil
}

func (worker *Checker) startRemote() error {
	_, err := worker.SourceProvider.GetRemote()
	worker.remoteEnabled = err == nil

	if !worker.remoteEnabled {
		worker.Logger.Info().Msg("Remote checker disabled")
		return nil
	}

	if worker.Config.MaxParallelRemoteChecks == 0 {
		worker.Config.MaxParallelRemoteChecks = runtime.NumCPU()

		worker.Logger.Info().
			Int("number_of_cpu", worker.Config.MaxParallelRemoteChecks).
			Msg("MaxParallelRemoteChecks is not configured, set it to the number of CPU")
	}

	// ==== Go remoteTriggerGetter ====
	worker.tomb.Go(worker.remoteTriggerGetter)
	worker.Logger.Info().Msg("Remote checker started")

	const maxParallelRemoteChecksMaxValue = 1024 * 8
	if worker.Config.MaxParallelRemoteChecks > maxParallelRemoteChecksMaxValue {
		return errors.New("MaxParallelRemoteChecks value is too large")
	}

	worker.Logger.Info().
		Int("number_of_checkers", worker.Config.MaxParallelRemoteChecks).
		Msg("Start parallel remote checkers")

	// ==== Go startTriggerToCheckGetter ====
	remoteTriggerIdsToCheckChan := worker.startTriggerToCheckGetter(worker.Database.GetRemoteTriggersToCheck, worker.Config.MaxParallelRemoteChecks)
	for i := 0; i < worker.Config.MaxParallelRemoteChecks; i++ {
		// ==== Go startTriggerHandler ====
		worker.tomb.Go(func() error {
			return worker.startTriggerHandler(remoteTriggerIdsToCheckChan, worker.Metrics.RemoteMetrics)
		})
	}

	return nil
}

func (worker *Checker) startVMSelect() error {
	_, err := worker.SourceProvider.GetVMSelect()
	worker.vmselectEnabled = err == nil

	if !worker.vmselectEnabled {
		worker.Logger.Info().Msg("VMSelect checker disabled")
		return nil
	}

	if worker.Config.MaxParallelVMSelectChecks == 0 {
		worker.Config.MaxParallelVMSelectChecks = runtime.NumCPU()

		worker.Logger.Info().
			Int("number_of_cpu", worker.Config.MaxParallelVMSelectChecks).
			Msg("MaxParallelRemoteChecks is not configured, set it to the number of CPU")
	}

	worker.tomb.Go(worker.vmselectTriggerGetter)
	worker.Logger.Info().Msg("VMSelect checker started")

	const maxParallelVMSelectChecksMaxValue = 1024 * 8
	if worker.Config.MaxParallelVMSelectChecks > maxParallelVMSelectChecksMaxValue {
		return errors.New("MaxParallelVMSelectChecks value is too large")
	}

	worker.Logger.Info().
		Int("number_of_checkers", worker.Config.MaxParallelVMSelectChecks).
		Msg("Start parallel remote checkers")

	vmselectTriggerIdsToCheckChan := worker.startTriggerToCheckGetter(worker.Database.GetVMSelectTriggersToCheck, worker.Config.MaxParallelVMSelectChecks)
	for i := 0; i < worker.Config.MaxParallelVMSelectChecks; i++ {
		worker.tomb.Go(func() error {
			return worker.startTriggerHandler(vmselectTriggerIdsToCheckChan, worker.Metrics.VMSelectMetrics)
		})
	}

	return nil
}

func (worker *Checker) startLocal() error {
	if worker.Config.MaxParallelChecks == 0 {
		worker.Config.MaxParallelChecks = runtime.NumCPU()
		worker.Logger.Info().
			Int("number_of_cpu", worker.Config.MaxParallelChecks).
			Msg("MaxParallelChecks is not configured, set it to the number of CPU")
	}

	worker.lastData = time.Now().UTC().Unix()

	if worker.Config.MetricEventPopBatchSize == 0 {
		worker.Config.MetricEventPopBatchSize = 100
	} else if worker.Config.MetricEventPopBatchSize < 0 {
		return errors.New("MetricEventPopBatchSize param less than zero")
	}

	subscribeMetricEventsParams := moira.SubscribeMetricEventsParams{
		BatchSize: worker.Config.MetricEventPopBatchSize,
		Delay:     worker.Config.MetricEventPopDelay,
	}

	metricEventsChannel, err := worker.Database.SubscribeMetricEvents(&worker.tomb, &subscribeMetricEventsParams)
	if err != nil {
		return err
	}

	worker.lazyTriggerIDs.Store(make(map[string]bool))
	worker.tomb.Go(worker.lazyTriggersWorker)

	worker.tomb.Go(worker.localTriggerGetter)

	// TODO: Proper config validation
	const maxParallelChecksMaxValue = 1024 * 8
	if worker.Config.MaxParallelChecks > maxParallelChecksMaxValue {
		return errors.New("MaxParallelChecks value is too large")
	}

	worker.Logger.Info().
		Int("number_of_checkers", worker.Config.MaxParallelChecks).
		Msg("Start parallel local checkers")

	localTriggerIdsToCheckChan := worker.startTriggerToCheckGetter(worker.Database.GetLocalTriggersToCheck, worker.Config.MaxParallelChecks)
	for i := 0; i < worker.Config.MaxParallelChecks; i++ {
		worker.tomb.Go(func() error {
			return worker.newMetricsHandler(metricEventsChannel)
		})
		worker.tomb.Go(func() error {
			return worker.startTriggerHandler(localTriggerIdsToCheckChan, worker.Metrics.LocalMetrics)
		})
	}

	worker.Logger.Info().Msg("Checking new events started")

	go func() {
		<-worker.tomb.Dying()
		worker.Logger.Info().Msg("Checking for new events stopped")
	}()

	worker.tomb.Go(func() error { return worker.checkMetricEventsChannelLen(metricEventsChannel) })
	worker.tomb.Go(worker.checkTriggersToCheckCount)

	return nil
}

func (worker *Checker) checkTriggersToCheckCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	var triggersToCheckCount, remoteTriggersToCheckCount int64
	var err error
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		case <-checkTicker.C:
			triggersToCheckCount, err = worker.Database.GetLocalTriggersToCheckCount()
			if err == nil {
				worker.Metrics.LocalMetrics.TriggersToCheckCount.Update(triggersToCheckCount)
			}
			if worker.remoteEnabled {
				remoteTriggersToCheckCount, err = worker.Database.GetRemoteTriggersToCheckCount()
				if err == nil {
					worker.Metrics.RemoteMetrics.TriggersToCheckCount.Update(remoteTriggersToCheckCount)
				}
			}
		}
	}
}

func (worker *Checker) checkMetricEventsChannelLen(ch <-chan *moira.MetricEvent) error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
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
