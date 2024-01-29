package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

func (manager *WorkerManager) startLocalMetricEvents() error {
	if manager.Config.MetricEventPopBatchSize < 0 {
		return errors.New("MetricEventPopBatchSize param was less than zero")
	}

	manager.Logger.Info().Msg("Starting local metric event handler")

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

	defaultLocalKey := moira.DefaultLocalCluster
	localConfig, ok := manager.Config.SourceCheckConfigs[defaultLocalKey]
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

func (manager *WorkerManager) newMetricsHandler(metricEventsChannel <-chan *moira.MetricEvent) error {
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			return nil
		}
		pattern := metricEvent.Pattern
		if manager.needHandlePattern(pattern) {
			if err := manager.handleMetricEvent(pattern); err != nil {
				manager.Logger.Error().
					Error(err).
					Msg("Failed to handle metricEvent")
			}
		}
	}
}

func (manager *WorkerManager) needHandlePattern(pattern string) bool {
	err := manager.PatternCache.Add(pattern, true, cache.DefaultExpiration)
	return err == nil
}

func (manager *WorkerManager) handleMetricEvent(pattern string) error {
	start := time.Now()
	defer manager.Metrics.MetricEventsHandleTime.UpdateSince(start)

	manager.lastData = time.Now().UTC().Unix()
	triggerIds, err := manager.Database.GetPatternTriggerIDs(pattern)
	if err != nil {
		return err
	}

	// Cleanup pattern and its metrics if this pattern doesn't match to any trigger
	if len(triggerIds) == 0 {
		if err := manager.Database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}

	manager.scheduleLocalTriggerIDsIfNeeded(triggerIds)
	return nil
}

func (manager *WorkerManager) scheduleLocalTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckTriggerIDs := manager.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		manager.Database.AddTriggersToCheck(moira.DefaultLocalCluster, needToCheckTriggerIDs) //nolint
	}
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
