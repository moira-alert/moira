package graphite

import "sync/atomic"

//CacheMetrics is a collection of metrics used in cache
type CacheMetrics struct {
	TotalMetricsReceived    Meter // TotalMetricsReceived metrics counter
	ValidMetricsReceived    Meter // ValidMetricsReceived metrics counter
	MatchingMetricsReceived Meter // MatchingMetricsReceived metrics counter
	MatchingTimer           Timer // MatchingTimer metrics timer
	SavingTimer             Timer // SavingTimer metrics timer
	BuildTreeTimer          Timer // BuildTreeTimer metrics timer
	TotalReceived           int64
	ValidReceived           int64
	MatchedReceived         int64
}

func (metrics *CacheMetrics) UpdateMetrics() {
	metrics.TotalMetricsReceived.Mark(atomic.SwapInt64(&metrics.TotalReceived, int64(0)))
	metrics.ValidMetricsReceived.Mark(atomic.SwapInt64(&metrics.ValidReceived, int64(0)))
	metrics.MatchingMetricsReceived.Mark(atomic.SwapInt64(&metrics.MatchedReceived, int64(0)))
}
