package patterns

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/metrics/graphite"
	"gopkg.in/tomb.v2"
)

type Matcher struct {
	logger         moira.Logger
	tomb           tomb.Tomb
	metrics        *graphite.FilterMetrics
	patternStorage *filter.PatternStorage
}

// NewMatcher creates pattern matcher
func NewMatcher(logger moira.Logger, metrics *graphite.FilterMetrics, patternsStorage *filter.PatternStorage) *Matcher {
	return &Matcher{
		logger:         logger,
		metrics:        metrics,
		patternStorage: patternsStorage,
	}
}

// Start spawns pattern matcher workers
func (m *Matcher) Start(workerCnt int, lineChan <-chan []byte) chan *moira.MatchedMetric {
	metricsChan := make(chan *moira.MatchedMetric, 16384)
	m.logger.Infof("starting %d pattern matcher workers", workerCnt)
	for i := 0; i < workerCnt; i++ {
		m.tomb.Go(func() error {
			return m.worker(lineChan, metricsChan)
		})
	}
	go func() {
		<-m.tomb.Dying()
		m.logger.Info("Stopping pattern matcher...")
		close(metricsChan)
		m.logger.Info("Moira pattern matcher stopped")
	}()

	return metricsChan
}

func (m *Matcher) worker(in <-chan []byte, out chan<- *moira.MatchedMetric) error {
	for line := range in {
		if m := m.patternStorage.ProcessIncomingMetric(line); m != nil {
			out <- m
		}
	}
	return nil
}

func (m *Matcher) checkNewMetricsChannelLen(channel <-chan *moira.MatchedMetric) error {
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
