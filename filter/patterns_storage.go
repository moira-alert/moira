package filter

import (
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// PatternStorage contains pattern tree
type PatternStorage struct {
	database                moira.Database
	metrics                 *metrics.FilterMetrics
	logger                  moira.Logger
	PatternIndex            atomic.Value
	SeriesByTagPatternIndex atomic.Value
}

// NewPatternStorage creates new PatternStorage struct
func NewPatternStorage(database moira.Database, metrics *metrics.FilterMetrics, logger moira.Logger) (*PatternStorage, error) {
	storage := &PatternStorage{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
	err := storage.Refresh()
	return storage, err
}

// Refresh builds pattern's indexes from redis data
func (storage *PatternStorage) Refresh() error {
	newPatterns, err := storage.database.AllowStale().GetPatterns()
	if err != nil {
		return err
	}

	seriesByTagPatterns := make(map[string][]TagSpec)
	patterns := make([]string, 0)
	for _, newPattern := range newPatterns {
		tagSpecs, err := ParseSeriesByTag(newPattern)
		if err == ErrNotSeriesByTag {
			patterns = append(patterns, newPattern)
		} else {
			seriesByTagPatterns[newPattern] = tagSpecs
		}
	}

	storage.PatternIndex.Store(NewPatternIndex(storage.logger, patterns))
	storage.SeriesByTagPatternIndex.Store(NewSeriesByTagPatternIndex(storage.logger, seriesByTagPatterns))
	return nil
}

// ProcessIncomingMetric validates, parses and matches incoming raw string
func (storage *PatternStorage) ProcessIncomingMetric(lineBytes []byte) *moira.MatchedMetric {
	storage.metrics.TotalMetricsReceived.Inc()
	count := storage.metrics.TotalMetricsReceived.Count()

	parsedMetric, err := ParseMetric(lineBytes)
	if err != nil {
		storage.logger.Infof("cannot parse input: %v", err)
		return nil
	}

	storage.metrics.ValidMetricsReceived.Inc()

	matchingStart := time.Now()
	matchedPatterns := storage.matchPatterns(parsedMetric)
	if count%10 == 0 {
		storage.metrics.MatchingTimer.UpdateSince(matchingStart)
	}
	if len(matchedPatterns) > 0 {
		storage.metrics.MatchingMetricsReceived.Inc()
		return &moira.MatchedMetric{
			Metric:             parsedMetric.Metric,
			Patterns:           matchedPatterns,
			Value:              parsedMetric.Value,
			Timestamp:          parsedMetric.Timestamp,
			RetentionTimestamp: parsedMetric.Timestamp,
			Retention:          60, //nolint
		}
	}
	return nil
}

func (storage *PatternStorage) matchPatterns(metric *ParsedMetric) []string {
	if metric.IsTagged() {
		seriesByTagPatternIndex := storage.SeriesByTagPatternIndex.Load().(*SeriesByTagPatternIndex)
		return seriesByTagPatternIndex.MatchPatterns(metric.Name, metric.Labels)
	}

	patternIndex := storage.PatternIndex.Load().(*PatternIndex)
	return patternIndex.MatchPatterns(metric.Name)
}
