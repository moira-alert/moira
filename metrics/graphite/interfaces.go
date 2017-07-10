package graphite

type MetricsMap interface {
	AddMetric(name, path string)
	GetMetric(name string) (Meter, bool)
}

type Meter interface {
	Count() int64
	Mark(int64)
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
}
