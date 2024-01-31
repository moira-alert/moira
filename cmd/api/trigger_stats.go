package main

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"gopkg.in/tomb.v2"
)

type triggerStats struct {
	tomb     tomb.Tomb
	metrics  *metrics.TriggersMetrics
	clusters []moira.ClusterKey
	database moira.Database
	logger   moira.Logger
}

func newTriggerStats(
	clusters []moira.ClusterKey,
	logger moira.Logger,
	database moira.Database,
	metricsRegistry metrics.Registry,
) *triggerStats {
	// sources := sourceProvider.GetAllSources()
	// clusters := make([]moira.ClusterKey, 0, len(sources))
	// for key := range sources {
	// 	clusters = append(clusters, key)
	// }

	return &triggerStats{
		logger:   logger,
		database: database,
		metrics:  metrics.NewTriggersMetrics(metricsRegistry, clusters),
		clusters: clusters,
	}
}

func (stats *triggerStats) start() {
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

func (stats *triggerStats) stop() error {
	stats.tomb.Kill(nil)
	return stats.tomb.Wait()
}

func (stats *triggerStats) checkTriggerCount() {
	triggersCount, err := stats.database.GetTriggerCount(stats.clusters)
	if err != nil {
		stats.logger.Warning().
			Error(err).
			Msg("Failed to fetch triggers count")
		return
	}

	for source, count := range triggersCount {
		stats.metrics.Mark(source, count)
	}
}
