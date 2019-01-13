package metricSource

// TriggerMetricsData represent collection of Main target timeseries and collection of additions targets timeseries
type TriggerMetricsData struct {
	Main       []*MetricData
	Additional []*MetricData
}

// MakeEmptyTriggerMetricsData just creates TriggerMetricsData with initialized empty fields
func MakeEmptyTriggerMetricsData() *TriggerMetricsData {
	return &TriggerMetricsData{
		Main:       make([]*MetricData, 0),
		Additional: make([]*MetricData, 0),
	}
}
