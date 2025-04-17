package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

const defaultRetention = 60

var (
	invalidRetentionsFormatErr = errors.New("Invalid retentions format, it is correct to write in the format 'retentions = timePerPoint:timeToStore, timePerPoint:timeToStore, ...'")
	invalidPatternFormatErr    = errors.New("Invalid pattern format, it is correct to write in the format 'pattern = regex'")
)

type retentionMatcher struct {
	pattern   *regexp.Regexp
	retention int
}

type retentionCacheItem struct {
	value     int
	timestamp int64
}

// Storage struct to store retention matchers.
type Storage struct {
	metrics         *metrics.FilterMetrics
	retentions      []retentionMatcher
	retentionsCache map[string]*retentionCacheItem
	metricsCache    map[string]*moira.MatchedMetric
	logger          moira.Logger
}

// NewCacheStorage create new Storage.
func NewCacheStorage(logger moira.Logger, metrics *metrics.FilterMetrics, reader io.Reader) (*Storage, error) {
	storage := &Storage{
		retentionsCache: make(map[string]*retentionCacheItem),
		metricsCache:    make(map[string]*moira.MatchedMetric),
		metrics:         metrics,
		logger:          logger,
	}

	if err := storage.buildRetentions(bufio.NewScanner(reader)); err != nil {
		return nil, err
	}

	return storage, nil
}

// EnrichMatchedMetric calculate retention and filter cached values.
func (storage *Storage) EnrichMatchedMetric(batch map[string]*moira.MatchedMetric, m *moira.MatchedMetric) {
	m.Retention = storage.getRetention(m)
	m.RetentionTimestamp = moira.RoundToNearestRetention(m.Timestamp, int64(m.Retention))

	if ex, ok := storage.metricsCache[m.Metric]; ok && ex.RetentionTimestamp == m.RetentionTimestamp && ex.Value == m.Value {
		return
	}

	if old, ok := batch[m.Metric]; ok && old.RetentionTimestamp != m.RetentionTimestamp {
		storage.logger.Warning().
			Int64("old_retention_timestamp", old.RetentionTimestamp).
			Int64("new_retention_timestamp", m.RetentionTimestamp).
			String("metric_name", m.Metric).
			Msg("Metric override")
	}

	storage.metricsCache[m.Metric] = m
	batch[m.Metric] = m
}

// getRetention returns first matched retention for metric.
func (storage *Storage) getRetention(m *moira.MatchedMetric) int {
	if item, ok := storage.retentionsCache[m.Metric]; ok && item.timestamp+60 > m.Timestamp {
		return item.value
	}

	for _, matcher := range storage.retentions {
		if matcher.pattern.MatchString(m.Metric) {
			storage.retentionsCache[m.Metric] = &retentionCacheItem{
				value:     matcher.retention,
				timestamp: m.Timestamp,
			}

			return matcher.retention
		}
	}

	return defaultRetention
}

func (storage *Storage) buildRetentions(retentionScanner *bufio.Scanner) error {
	storage.retentions = make([]retentionMatcher, 0, 100)

	for retentionScanner.Scan() {
		patternLine := retentionScanner.Text()
		if strings.HasPrefix(patternLine, "#") || strings.Count(patternLine, "=") < 1 {
			continue
		}

		_, after, found := strings.Cut(patternLine, "=")
		if !found {
			storage.logger.Error().
				Error(invalidPatternFormatErr).
				String("pattern_line", patternLine).
				Msg("Invalid pattern format")

			continue
		}

		patternString := strings.TrimSpace(after)

		pattern, err := regexp.Compile(patternString)
		if err != nil {
			return fmt.Errorf("failed to compile regexp pattern '%s': %w", patternString, err)
		}

		retentionScanner.Scan()
		retentionsLine := retentionScanner.Text()
		splitted := strings.Split(retentionsLine, "=")

		if len(splitted) < 2 { //nolint
			storage.logger.Error().
				Error(invalidRetentionsFormatErr).
				String("pattern_line", patternLine).
				String("retentions_line", retentionsLine).
				Msg("Invalid retentions format")

			continue
		}

		retentions := strings.TrimSpace(splitted[1])

		retention, err := rawRetentionToSeconds(retentions[0:strings.Index(retentions, ":")])
		if err != nil {
			return fmt.Errorf("failed to convert raw retentions '%s' to seconds: %w", retentions, err)
		}

		storage.retentions = append(storage.retentions, retentionMatcher{
			pattern:   pattern,
			retention: retention,
		})
	}

	return retentionScanner.Err()
}

func rawRetentionToSeconds(rawRetention string) (int, error) {
	retention, err := strconv.Atoi(rawRetention)
	if err == nil {
		return retention, nil
	}

	multiplier := 1

	switch {
	case strings.HasSuffix(rawRetention, "m"):
		multiplier = 60
	case strings.HasSuffix(rawRetention, "h"):
		multiplier = 60 * 60 //nolint
	case strings.HasSuffix(rawRetention, "d"):
		multiplier = 60 * 60 * 24 //nolint
	case strings.HasSuffix(rawRetention, "w"):
		multiplier = 60 * 60 * 24 * 7 //nolint
	case strings.HasSuffix(rawRetention, "y"):
		multiplier = 60 * 60 * 24 * 365 //nolint
	}

	retention, err = strconv.Atoi(rawRetention[0 : len(rawRetention)-1])
	if err != nil {
		return 0, err
	}

	return retention * multiplier, nil
}
