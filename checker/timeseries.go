package checker

type TargetTimeSeries struct {
	OtherTargetsNames map[string]string
	TimeSeries        map[int][]TimeSeries
}
