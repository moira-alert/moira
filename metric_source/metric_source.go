package metricSource

// MetricSource implements graphite metrics source abstraction
type MetricSource interface {
	Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (FetchResult, error)
}

// FetchResult implements moira metric sources fetching result format
type FetchResult interface {
	GetMetricsData() []*MetricData
	GetPatterns() ([]string, error)
	GetPatterMetrics() ([]string, error)
}
