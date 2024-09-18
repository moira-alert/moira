package stats

import (
	"gopkg.in/tomb.v2"
)

// StatsReporter represents an interface for objects that report statistics.
type StatsReporter interface {
	StartReport(stop <-chan struct{})
}

type statsManager struct {
	tomb      tomb.Tomb
	reporters []StatsReporter
}

// NewStatsManager creates a new statsManager instance with the given StatsReporters.
func NewStatsManager(reporters ...StatsReporter) *statsManager {
	return &statsManager{
		reporters: reporters,
	}
}

// Start starts reporting statistics for all registered StatsReporters.
func (manager *statsManager) Start() {
	for _, reporter := range manager.reporters {
		reporter := reporter

		manager.tomb.Go(func() error {
			reporter.StartReport(manager.tomb.Dying())
			return nil
		})
	}
}

// Stop stops all reporting activities and waits for the completion.
func (manager *statsManager) Stop() error {
	manager.tomb.Kill(nil)
	return manager.tomb.Wait()
}
