package handler

const allMetricsPattern = ".*"

// default values for event router.
const (
	// default values for middleware.Paginate.
	eventDefaultPage = 0
	eventDefaultSize = -1

	// default values for middleware.DateRange.
	eventDefaultFrom = "-3hour"
	eventDefaultTo   = "now"

	// default value for middleware.MetricProvider.
	eventDefaultMetric = allMetricsPattern
)
