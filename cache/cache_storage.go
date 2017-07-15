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
	"time"
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

// CacheStorage struct to store retention matchers
type CacheStorage struct {
	database        moira.Database
	metrics         *graphite.CacheMetrics
	retentions      []retentionMatcher
	retentionsCache map[string]*retentionCacheItem
	metricsCache    map[string]*moira.MatchedMetric
}

// NewCacheStorage create new CacheStorage
func NewCacheStorage(database moira.Database, metrics *graphite.CacheMetrics, retentionConfigFileName string) (*CacheStorage, error) {
	retentionConfigFile, err := os.Open(retentionConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("Error open retentions file [%s]: %s", retentionConfigFileName, err.Error())
	}

	storage := &CacheStorage{
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

// ProcessMatchedMetrics make buffer of metrics and save it
func (storage *CacheStorage) ProcessMatchedMetrics(ch chan *moira.MatchedMetric, save func(map[string]*moira.MatchedMetric)) {
	buffer := make(map[string]*moira.MatchedMetric)
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return
			}

			storage.EnrichMatchedMetric(buffer, m)

			if len(buffer) < 10 {
				continue
			}
			break
		case <-time.After(time.Second):
			break
		}
		if len(buffer) == 0 {
			continue
		}
		timer := time.Now()
		save(buffer)
		storage.metrics.SavingTimer.UpdateSince(timer)
		buffer = make(map[string]*moira.MatchedMetric)
	}
}

// EnrichMatchedMetric calculate retention and filter cached values
func (storage *CacheStorage) EnrichMatchedMetric(buffer map[string]*moira.MatchedMetric, m *moira.MatchedMetric) {
	m.Retention = storage.GetRetention(m)
	m.RetentionTimestamp = roundToNearestRetention(m.Timestamp, int64(m.Retention))
	if ex, ok := storage.metricsCache[m.Metric]; ok && ex.RetentionTimestamp == m.RetentionTimestamp && ex.Value == m.Value {
		return
	}
	storage.metricsCache[m.Metric] = m
	buffer[m.Metric] = m
}

// SavePoints saving matched metrics to DB
func (storage *CacheStorage) SavePoints(buffer map[string]*moira.MatchedMetric) error {

	if err := storage.database.SaveMetrics(buffer); err != nil {
		return err
	}

	return nil
}

// GetRetention returns first matched retention for metric
func (storage *CacheStorage) GetRetention(m *moira.MatchedMetric) int {
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

func (storage *CacheStorage) buildRetentions(retentionScanner *bufio.Scanner) error {
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
