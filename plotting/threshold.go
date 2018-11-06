package plotting

import (
	"math"
	"time"

	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
)

const (
	// ThresholdSerie is a name that indicates threshold
	ThresholdSerie = "threshold"
	// InvertedThresholdGap is max allowed (area between thresholds)^(-1)
	InvertedThresholdGap = 16
)

// threshold represents threshold parameters
type threshold struct {
	thresholdType string
	xCoordinate   float64
	yCoordinate   float64
}

// getThresholdSeriesList returns collection of thresholds and annotations
func getThresholdSeriesList(trigger *moira.Trigger, theme moira.PlotTheme, limits plotLimits) []chart.Series {
	thresholdSeriesList := make([]chart.Series, 0)
	if trigger.TriggerType == moira.ExpressionTrigger {
		return thresholdSeriesList
	}
	plotThresholds := generateThresholds(trigger, limits)
	for _, plotThreshold := range plotThresholds {
		thresholdSeriesList = append(thresholdSeriesList, plotThreshold.generateThresholdSeries(theme, limits))
		thresholdSeriesList = append(thresholdSeriesList, plotThreshold.generateAnnotationSeries(theme))
	}
	return thresholdSeriesList
}

// generateThresholds returns thresholds available for plot
func generateThresholds(trigger *moira.Trigger, limits plotLimits) []*threshold {
	// TODO: cover cases with negative warn & error values
	// TODO: cover cases with out-of-range thresholds (no annotations required)
	timePoint := float64(limits.to.UnixNano())
	thresholds := make([]*threshold, 0)
	switch trigger.ErrorValue {
	case nil:
		if trigger.WarnValue != nil && !(*trigger.WarnValue < 0) {
			if limits.formsSetContaining(*trigger.WarnValue) {
				warnThreshold := &threshold{
					thresholdType: "WARN",
					xCoordinate:   timePoint,
					yCoordinate:   *trigger.WarnValue,
				}
				if trigger.TriggerType == moira.RisingTrigger {
					warnThreshold.yCoordinate = limits.highest - *trigger.WarnValue
				}
				thresholds = append(thresholds, warnThreshold)
			}
		}
	default:
		if !(*trigger.ErrorValue < 0) {
			if limits.formsSetContaining(*trigger.ErrorValue) {
				errThreshold := &threshold{
					thresholdType: "ERROR",
					xCoordinate:   timePoint,
					yCoordinate:   *trigger.ErrorValue,
				}
				if trigger.TriggerType == moira.RisingTrigger {
					errThreshold.yCoordinate = limits.highest - *trigger.ErrorValue
				}
				thresholds = append(thresholds, errThreshold)
			}
		}
		if trigger.WarnValue != nil {
			deltaLimits := math.Abs(limits.highest - limits.lowest)
			deltaThresholds := math.Abs(*trigger.ErrorValue - *trigger.WarnValue)
			if !(*trigger.WarnValue < 0) && deltaThresholds > (deltaLimits/InvertedThresholdGap) {
				if limits.formsSetContaining(*trigger.WarnValue) {
					warnThreshold := &threshold{
						thresholdType: "WARN",
						xCoordinate:   timePoint,
						yCoordinate:   *trigger.WarnValue,
					}
					if trigger.TriggerType == moira.RisingTrigger {
						warnThreshold.yCoordinate = limits.highest - *trigger.WarnValue
					}
					thresholds = append(thresholds, warnThreshold)
				}
			}
		}
	}
	return thresholds
}

// generateThresholdSeries returns threshold series
func (threshold *threshold) generateThresholdSeries(theme moira.PlotTheme, limits plotLimits) chart.TimeSeries {
	thresholdSeries := chart.TimeSeries{
		Name:    ThresholdSerie,
		Style:   theme.GetThresholdStyle(threshold.thresholdType),
		XValues: []time.Time{limits.from, limits.to},
		YValues: []float64{},
	}
	for j := 0; j < len(thresholdSeries.XValues); j++ {
		thresholdSeries.YValues = append(thresholdSeries.YValues, threshold.yCoordinate)
	}
	return thresholdSeries
}

// generateAnnotationSeries returns threshold annotation series
func (threshold *threshold) generateAnnotationSeries(theme moira.PlotTheme) chart.AnnotationSeries {
	annotationSeries := chart.AnnotationSeries{
		Annotations: []chart.Value2{
			{
				Label:  threshold.thresholdType,
				XValue: threshold.xCoordinate,
				YValue: threshold.yCoordinate,
				Style:  theme.GetAnnotationStyle(threshold.thresholdType),
			},
		},
	}
	return annotationSeries
}
