package filter

import (
	"sync/atomic"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/metrics"
)

// PatternStorage contains pattern tree
type PatternStorage struct {
	database                moira2.Database
	metrics                 *metrics.FilterMetrics
	logger                  moira2.Logger
	PatternIndex            atomic.Value
	SeriesByTagPatternIndex atomic.Value
}

// NewPatternStorage creates new PatternStorage struct
func NewPatternStorage(database moira2.Database, metrics *metrics.FilterMetrics, logger moira2.Logger) (*PatternStorage, error) {
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
	storage.SeriesByTagPatternIndex.Store(NewSeriesByTagPatternIndex(seriesByTagPatterns))
	return nil
}

// ProcessIncomingMetric validates, parses and matches incoming raw string
func (storage *PatternStorage) ProcessIncomingMetric(lineBytes []byte) *moira2.MatchedMetric {
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
		return &moira2.MatchedMetric{
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
	seriesByTagPatternIndex := storage.SeriesByTagPatternIndex.Load().(*SeriesByTagPatternIndex)

	matchedPatterns := make([]string, 0)
	matchedPatterns = append(matchedPatterns, patternIndex.MatchPatterns(metric.Name)...)
	matchedPatterns = append(matchedPatterns, seriesByTagPatternIndex.MatchPatterns(metric.Name, metric.Labels)...)
	return matchedPatterns
}
