package metrics

import (
	"strings"
	"sync"

	"github.com/moira-alert/moira/metrics/graphite"
)

// TimerMap is realization of metrics map of type Timer
type TimerMap struct {
	metrics map[string]Timer
	addLock sync.Mutex
	prefix  string
}

// newTimerMap create empty Meter map
func newTimerMap(prefix string) *TimerMap {
	return &TimerMap{
		metrics: make(map[string]Timer),
		prefix:  prefix,
	}
}

// GetOrAdd gets timer and, if it does not exists, add it do map
func (timerMap *TimerMap) GetOrAdd(name, graphitePath string) graphite.Timer {
	if _, ok := timerMap.metrics[name]; !ok {
		timerMap.addLock.Lock()
		defer timerMap.addLock.Unlock()
		if _, ok := timerMap.metrics[name]; !ok {
			newMetricsMap := make(map[string]Timer, len(timerMap.metrics)+1)
			for k, v := range timerMap.metrics {
				newMetricsMap[k] = v
			}
			newMetricsMap[name] = *registerTimer(metricNameWithPrefix(timerMap.prefix, strings.Replace(graphitePath, "-", "_", -1)))
			timerMap.metrics = newMetricsMap
		}
	}
	value := timerMap.metrics[name]
	return &value
}
