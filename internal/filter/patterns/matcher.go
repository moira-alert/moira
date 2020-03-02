package patterns

import (
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/filter"
	"github.com/moira-alert/moira/internal/metrics"
	"gopkg.in/tomb.v2"
)

// Matcher checks metrics against known patterns
type Matcher struct {
	logger         moira2.Logger
	tomb           tomb.Tomb
	metrics        *metrics.FilterMetrics
	patternStorage *filter.PatternStorage
}

// NewMatcher creates pattern matcher
func NewMatcher(logger moira2.Logger, metrics *metrics.FilterMetrics, patternsStorage *filter.PatternStorage) *Matcher {
	return &Matcher{
		logger:         logger,
		metrics:        metrics,
		patternStorage: patternsStorage,
	}
}

// Start spawns pattern matcher workers
func (m *Matcher) Start(matchersCount int, lineChan <-chan []byte) chan *moira2.MatchedMetric {
	matchedMetricsChan := make(chan *moira2.MatchedMetric, 16384)
	m.logger.Infof("Start %d pattern matcher workers", matchersCount)
	for i := 0; i < matchersCount; i++ {
		m.tomb.Go(func() error {
			return m.worker(lineChan, matchedMetricsChan)
		})
	}
	go func() {
		<-m.tomb.Dying()
		m.logger.Info("Stopping pattern matcher...")
		close(matchedMetricsChan)
		m.logger.Info("Moira pattern matcher stopped")
	}()

	m.tomb.Go(func() error { return m.checkNewMetricsChannelLen(matchedMetricsChan) })
	return matchedMetricsChan
}

func (m *Matcher) worker(metricsChan <-chan []byte, matchedMetricsChan chan<- *moira2.MatchedMetric) error {
	for line := range metricsChan {
		if metric := m.patternStorage.ProcessIncomingMetric(line); metric != nil {
			matchedMetricsChan <- metric
		}
	}
	return nil
}

func (m *Matcher) checkNewMetricsChannelLen(channel <-chan *moira2.MatchedMetric) error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-m.tomb.Dying():
			return nil
		case <-checkTicker.C:
			m.metrics.MetricChannelLen.Update(int64(len(channel)))
		}
	}
}
