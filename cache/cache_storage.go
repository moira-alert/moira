package cache

import (
	"bufio"
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	database        moira.Database
	metrics         *graphite.CacheMetrics
	retentions      []retentionMatcher
	retentionsCache map[string]*retentionCacheItem
	metricsCache    map[string]*moira.MatchedMetric
}

// NewCacheStorage create new Storage
func NewCacheStorage(database moira.Database, metrics *graphite.CacheMetrics, retentionConfigFileName string) (*Storage, error) {
	retentionConfigFile, err := os.Open(retentionConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("Error open retentions file [%s]: %s", retentionConfigFileName, err.Error())
	}

	storage := &Storage{
		retentionsCache: make(map[string]*retentionCacheItem),
		metricsCache:    make(map[string]*moira.MatchedMetric),
		database:        database,
		metrics:         metrics,
	}

	if err := storage.buildRetentions(bufio.NewScanner(retentionConfigFile)); err != nil {
		return nil, err
	}
	return storage, nil
}

// EnrichMatchedMetric calculate retention and filter cached values
func (storage *Storage) EnrichMatchedMetric(buffer map[string]*moira.MatchedMetric, m *moira.MatchedMetric) {
	m.Retention = storage.GetRetention(m)
	m.RetentionTimestamp = roundToNearestRetention(m.Timestamp, int64(m.Retention))
	if ex, ok := storage.metricsCache[m.Metric]; ok && ex.RetentionTimestamp == m.RetentionTimestamp && ex.Value == m.Value {
		return
	}
	storage.metricsCache[m.Metric] = m
	buffer[m.Metric] = m
}

// GetRetention returns first matched retention for metric
func (storage *Storage) GetRetention(m *moira.MatchedMetric) int {
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
		line := retentionScanner.Text()
		if strings.HasPrefix(line, "#") || strings.Count(line, "=") != 1 {
			continue
		}

		pattern, err := regexp.Compile(strings.TrimSpace(strings.Split(line, "=")[1]))
		if err != nil {
			return err
		}

		retentionScanner.Scan()
		line = retentionScanner.Text()
		retentions := strings.TrimSpace(strings.Split(line, "=")[1])
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
