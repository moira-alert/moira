package metricSource

import "fmt"

// TriggerMetricsData represent collection of Main target timeseries and collection of additions targets timeseries
type TriggerMetricsData struct {
	Main       []*MetricData
	Additional []*MetricData
}

// NewTriggerMetricsData is a constructor function that creates TriggerMetricsData with initialized empty fields
func NewTriggerMetricsData() *TriggerMetricsData {
	return &TriggerMetricsData{
		Main:       make([]*MetricData, 0),
		Additional: make([]*MetricData, 0),
	}
}

// MakeTriggerMetricsData just creates TriggerMetricsData with given main and additional metrics data
func MakeTriggerMetricsData(main []*MetricData, additional []*MetricData) *TriggerMetricsData {
	return &TriggerMetricsData{
		Main:       main,
		Additional: additional,
	}
}

// GetMainTargetName just gets triggers main targets name (always is 't1')
func (*TriggerMetricsData) GetMainTargetName() string {
	return "t1"
}

// GetAdditionalTargetName gets triggers additional target name
func (*TriggerMetricsData) GetAdditionalTargetName(targetIndex int) string {
	return fmt.Sprintf("t%v", targetIndex+2)
}

// HasOnlyWildcards checks given targetTimeSeries for only wildcards
func (triggerTimeSeries *TriggerMetricsData) HasOnlyWildcards() bool {
	for _, timeSeries := range triggerTimeSeries.Main {
		if !timeSeries.Wildcard {
			return false
		}
	}
	return true
}
