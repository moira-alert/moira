package filter

import (
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite"
)

// PatternStorage contains pattern tree
type PatternStorage struct {
	database         moira.Database
	metrics          *graphite.FilterMetrics
	logger           moira.Logger
	PatternIndex     atomic.Value
	SeriesByTagIndex atomic.Value
}

// NewPatternStorage creates new PatternStorage struct
func NewPatternStorage(database moira.Database, metrics *graphite.FilterMetrics, logger moira.Logger) (*PatternStorage, error) {
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
	newPatterns, err := storage.database.GetPatterns()
	if err != nil {
		return err
	}

	seriesByTagPatterns := make(map[string][]TagSpec, 0)
	patterns := make([]string, 0)
	for _, newPattern := range newPatterns {
		tagSpecs, err := ParseSeriesByTag(newPattern)
		if err == ErrNotSeriesByTag {
			patterns = append(patterns, newPattern)
		} else {
			seriesByTagPatterns[newPattern] = tagSpecs
		}
	}

	storage.PatternIndex.Store(NewPatternIndex(patterns))
	storage.SeriesByTagIndex.Store(NewSeriesByTagIndex(seriesByTagPatterns))
	return nil
}

// ProcessIncomingMetric validates, parses and matches incoming raw string
func (storage *PatternStorage) ProcessIncomingMetric(lineBytes []byte) *moira.MatchedMetric {
	storage.metrics.TotalMetricsReceived.Inc(1)
	count := storage.metrics.TotalMetricsReceived.Count()

	parsedMetric, err := ParseMetric(lineBytes)
	if err != nil {
		storage.logger.Infof("cannot parse input: %v", err)
		return nil
	}

	storage.metrics.ValidMetricsReceived.Inc(1)

	matchingStart := time.Now()
	matchedPatterns := storage.matchPatterns(parsedMetric)
	if count%10 == 0 {
		storage.metrics.MatchingTimer.UpdateSince(matchingStart)
	}
	if len(matchedPatterns) > 0 {
		storage.metrics.MatchingMetricsReceived.Inc(1)
		return &moira.MatchedMetric{
			Metric:             parsedMetric.Metric,
			Patterns:           matchedPatterns,
			Value:              parsedMetric.Value,
			Timestamp:          parsedMetric.Timestamp,
			RetentionTimestamp: parsedMetric.Timestamp,
			Retention:          60,
		}
	}
	return nil
}

func (storage *PatternStorage) matchPatterns(metric *ParsedMetric) []string {
	patternIndex := storage.PatternIndex.Load().(*PatternIndex)
	seriesByTagIndex := storage.SeriesByTagIndex.Load().(*SeriesByTagIndex)

	matchedPatterns := make([]string, 0)
	matchedPatterns = append(matchedPatterns, patternIndex.MatchPatterns(metric.Name)...)
	matchedPatterns = append(matchedPatterns, seriesByTagIndex.MatchPatterns(metric.Name, metric.Labels)...)
	return matchedPatterns
}
