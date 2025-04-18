package filter

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/clock"

	lrucache "github.com/hashicorp/golang-lru/v2"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// PatternStorageConfig defines the configuration for pattern storage.
type PatternStorageConfig struct {
	// PatternMatchingCacheSize determines the size of the pattern matching cache.
	PatternMatchingCacheSize int
}

type patternMatchingCacheItem struct {
	nameTagValue    string
	matchingHandler MatchingHandler
}

// PatternStorage contains pattern tree.
type PatternStorage struct {
	database                moira.Database
	metrics                 *metrics.FilterMetrics
	clock                   moira.Clock
	logger                  moira.Logger
	PatternIndex            atomic.Value
	SeriesByTagPatternIndex atomic.Value
	compatibility           Compatibility
	patternMatchingCache    *lrucache.Cache[string, *patternMatchingCacheItem]
}

// NewPatternStorage creates new PatternStorage struct.
func NewPatternStorage(
	cfg PatternStorageConfig,
	database moira.Database,
	metrics *metrics.FilterMetrics,
	logger moira.Logger,
	compatibility Compatibility,
) (*PatternStorage, error) {
	patternMatchingCache, err := lrucache.New[string, *patternMatchingCacheItem](cfg.PatternMatchingCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create new lru pattern matching cache: %w", err)
	}

	storage := &PatternStorage{
		database:             database,
		metrics:              metrics,
		logger:               logger,
		clock:                clock.NewSystemClock(),
		compatibility:        compatibility,
		patternMatchingCache: patternMatchingCache,
	}

	if err = storage.Refresh(); err != nil {
		return nil, fmt.Errorf("failed to refresh pattern storage: %w", err)
	}

	return storage, nil
}

// Refresh builds pattern's indexes from redis data.
func (storage *PatternStorage) Refresh() error {
	newPatterns, err := storage.database.GetPatterns()
	if err != nil {
		return err
	}

	seriesByTagPatterns := make(map[string][]TagSpec)
	patterns := make([]string, 0)

	for _, newPattern := range newPatterns {
		tagSpecs, err := ParseSeriesByTag(newPattern)
		if errors.Is(err, ErrNotSeriesByTag) {
			patterns = append(patterns, newPattern)
		} else {
			seriesByTagPatterns[newPattern] = tagSpecs
		}
	}

	storage.PatternIndex.Store(NewPatternIndex(
		storage.logger,
		patterns,
		storage.compatibility,
	))

	storage.SeriesByTagPatternIndex.Store(NewSeriesByTagPatternIndex(
		storage.logger,
		seriesByTagPatterns,
		storage.compatibility,
		storage.patternMatchingCache,
		storage.metrics,
	))

	return nil
}

// ProcessIncomingMetric validates, parses and matches incoming raw string.
func (storage *PatternStorage) ProcessIncomingMetric(lineBytes []byte, maxTTL time.Duration) *moira.MatchedMetric {
	storage.metrics.TotalMetricsReceived.Inc()
	count := storage.metrics.TotalMetricsReceived.Count()

	parsedMetric, err := ParseMetric(lineBytes)
	if err != nil {
		storage.logger.Info().
			Error(err).
			Msg("Cannot parse input")

		return nil
	}

	if parsedMetric.IsExpired(maxTTL, storage.clock.NowUTC()) {
		storage.logger.Debug().
			String(moira.LogFieldNameMetricName, parsedMetric.Name).
			String(moira.LogFieldNameMetricTimestamp, fmt.Sprint(parsedMetric.Timestamp)).
			Msg("Metric is not in the window from maxTTL")

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

	storage.logger.Debug().
		String("metric", parsedMetric.Metric).
		Msg("Metric is not matched with prefix tree")

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
