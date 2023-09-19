package main

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"gopkg.in/tomb.v2"
)

type triggerStats struct {
	tomb     tomb.Tomb
	metrics  *metrics.TriggersMetrics
	database moira.Database
	logger   moira.Logger
}

func newTriggerStats(
	logger moira.Logger,
	database moira.Database,
	metricsRegistry metrics.Registry,
) *triggerStats {
	return &triggerStats{
		logger:   logger,
		database: database,
		metrics:  metrics.NewTriggersMetrics(metricsRegistry),
	}
}

func (stats *triggerStats) Start() {
	stats.tomb.Go(stats.startCheckingTriggerCount)
}

func (stats *triggerStats) startCheckingTriggerCount() error {
	checkTicker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-stats.tomb.Dying():
			return nil

		case <-checkTicker.C:
			stats.checkTriggerCount()
		}
	}
}

func (stats *triggerStats) Stop() error {
	stats.tomb.Kill(nil)
	return stats.tomb.Wait()
}

func (stats *triggerStats) checkTriggerCount() {
	triggersCount, err := stats.database.GetTriggerCount()
	if err != nil {
		stats.logger.Warning().
			Error(err).
			Msg("Failed to fetch triggers count")
		return
	}

	for source, count := range triggersCount {
		stats.metrics.Mark(source, count)
		stats.logger.Debug().Msg(fmt.Sprintf("source: %s, count: %d", string(source), count))
	}
}
