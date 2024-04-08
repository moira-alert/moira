package stats

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type triggerStats struct {
	metrics  *metrics.TriggersMetrics
	database moira.Database
	logger   moira.Logger
	clusters []moira.ClusterKey
}

// NewTriggerStats creates and initializes a new triggerStats object.
func NewTriggerStats(
	metricsRegistry metrics.Registry,
	database moira.Database,
	logger moira.Logger,
	clusters []moira.ClusterKey,
) *triggerStats {
	return &triggerStats{
		logger:   logger,
		database: database,
		metrics:  metrics.NewTriggersMetrics(metricsRegistry, clusters),
		clusters: clusters,
	}
}

// StartReport starts reporting statistics about triggers.
func (stats *triggerStats) StartReport(stop <-chan struct{}) {
	checkTicker := time.NewTicker(time.Minute)
	defer checkTicker.Stop()

	stats.logger.Info().Msg("Start trigger statistics reporter")

	for {
		select {
		case <-stop:
			stats.logger.Info().Msg("Stop trigger statistics reporter")
			return

		case <-checkTicker.C:
			stats.checkTriggerCount()
		}
	}
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
