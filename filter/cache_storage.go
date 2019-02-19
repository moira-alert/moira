package filter

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite"
)

var defaultRetention = 60

type retentionMatcher struct {
	pattern   *regexp.Regexp
	retention int
}

type retentionCacheItem struct {
	value     int
	timestamp int64
}

// Storage struct to store retention matchers
type Storage struct {
	metrics         *graphite.FilterMetrics
	retentions      []retentionMatcher
	retentionsCache map[string]*retentionCacheItem
	metricsCache    map[string]*moira.MatchedMetric
	logger          moira.Logger
}

// NewCacheStorage create new Storage
func NewCacheStorage(logger moira.Logger, metrics *graphite.FilterMetrics, reader io.Reader) (*Storage, error) {
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

// EnrichMatchedMetric calculate retention and filter cached values
func (storage *Storage) EnrichMatchedMetric(batch map[string]*moira.MatchedMetric, m *moira.MatchedMetric) {
	m.Retention = storage.getRetention(m)
	m.RetentionTimestamp = roundToNearestRetention(m.Timestamp, int64(m.Retention))
	if ex, ok := storage.metricsCache[m.Metric]; ok && ex.RetentionTimestamp == m.RetentionTimestamp && ex.Value == m.Value {
		return
	}
	storage.metricsCache[m.Metric] = m
	batch[m.Metric] = m
}

// getRetention returns first matched retention for metric
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
		line1 := retentionScanner.Text()
		if strings.HasPrefix(line1, "#") || strings.Count(line1, "=") != 1 {
			continue
		}

		patternString := strings.TrimSpace(strings.Split(line1, "=")[1])
		pattern, err := regexp.Compile(patternString)
		if err != nil {
			return err
		}

		retentionScanner.Scan()
		line2 := retentionScanner.Text()
		splitted := strings.Split(line2, "=")

		if len(splitted) < 2 {
			storage.logger.Errorf("Invalid pattern found: '%s'", patternString)
			continue
		}

		retentions := strings.TrimSpace(splitted[1])
		retention, err := rawRetentionToSeconds(retentions[0:strings.Index(retentions, ":")])
		if err != nil {
			return err
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
		multiplier = 60 * 60
	case strings.HasSuffix(rawRetention, "d"):
		multiplier = 60 * 60 * 24
	case strings.HasSuffix(rawRetention, "w"):
		multiplier = 60 * 60 * 24 * 7
	case strings.HasSuffix(rawRetention, "y"):
		multiplier = 60 * 60 * 24 * 365
	}

	retention, err = strconv.Atoi(rawRetention[0 : len(rawRetention)-1])
	if err != nil {
		return 0, err
	}

	return retention * multiplier, nil
}

func roundToNearestRetention(ts, retention int64) int64 {
	return (ts + retention/2) / retention * retention
}
