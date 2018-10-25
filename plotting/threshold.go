package plotting

import (
	"github.com/moira-alert/moira"
	"math"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	// ThresholdSerie is a name that indicates threshold
	ThresholdSerie = "threshold"
	// InvertedThresholdGap is max allowed (area between thresholds)^(-1)
	InvertedThresholdGap = 16
)

// threshold represents threshold parameters
type threshold struct {
	title     string
	value     float64
	timePoint float64
	color     string
	growTo    int
}

// getThresholdSeriesList returns collection of thresholds and annotations
func getThresholdSeriesList(trigger *moira.Trigger, limits plotLimits, theme *plotTheme) ([]chart.Series, bool) {
	thresholdSeriesList := make([]chart.Series, 0)
	if trigger.TriggerType == moira.ExpressionTrigger {
		return thresholdSeriesList, false
	}
	plotThresholds := generateThresholds(trigger, limits, theme)
	for _, plotThreshold := range plotThresholds {
		thresholdSeriesList = append(thresholdSeriesList, plotThreshold.generateThresholdSeries(limits))
		thresholdSeriesList = append(thresholdSeriesList, plotThreshold.generateAnnotationSeries(limits, theme.font))
	}
	return thresholdSeriesList, true
}

// generateThresholds returns thresholds available for plot
func generateThresholds(trigger *moira.Trigger, limits plotLimits, theme *plotTheme) []*threshold {
	// TODO: cover cases with negative warn & error values
	// TODO: cover cases with out-of-range thresholds (no annotations required)
	timePoint := float64(limits.to.UnixNano())
	thresholds := make([]*threshold, 0)
	switch trigger.ErrorValue {
	case nil:
		if trigger.WarnValue != nil && !(*trigger.WarnValue < 0) {
			if limits.formsSetContaining(*trigger.WarnValue) {
				warnThreshold := &threshold{
					title:     "WARN",
					value:     *trigger.WarnValue,
					timePoint: timePoint,
					color:     theme.warnThresholdColor,
					growTo:    9,
				}
				if trigger.TriggerType == moira.RisingTrigger {
					warnThreshold.value = limits.highest - *trigger.WarnValue
				}
				thresholds = append(thresholds, warnThreshold)
			}
		}
	default:
		if !(*trigger.ErrorValue < 0) {
			if limits.formsSetContaining(*trigger.ErrorValue) {
				errThreshold := &threshold{
					title:     "ERROR",
					value:     *trigger.ErrorValue,
					timePoint: timePoint,
					color:     theme.errorThresholdColor,
					growTo:    0,
				}
				if trigger.TriggerType == moira.RisingTrigger {
					errThreshold.value = limits.highest - *trigger.ErrorValue
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
						title:     "WARN",
						value:     *trigger.WarnValue,
						timePoint: timePoint,
						color:     theme.warnThresholdColor,
						growTo:    9,
					}
					if trigger.TriggerType == moira.RisingTrigger {
						warnThreshold.value = limits.highest - *trigger.WarnValue
					}
					thresholds = append(thresholds, warnThreshold)
				}
			}
		}
	}
	return thresholds
}

// generateThresholdSeries returns threshold series
func (threshold *threshold) generateThresholdSeries(limits plotLimits) chart.TimeSeries {
	thresholdSeries := chart.TimeSeries{
		Name: ThresholdSerie,
		Style: chart.Style{
			Show:        true,
			StrokeWidth: 1,
			StrokeColor: drawing.ColorFromHex(threshold.color).WithAlpha(90),
			FillColor:   drawing.ColorFromHex(threshold.color).WithAlpha(20),
		},

		XValues: []time.Time{limits.from, limits.to},
		YValues: []float64{},
	}
	for j := 0; j < len(thresholdSeries.XValues); j++ {
		thresholdSeries.YValues = append(thresholdSeries.YValues, threshold.value)
	}
	return thresholdSeries
}

// generateAnnotationSeries returns threshold annotation series
func (threshold *threshold) generateAnnotationSeries(limits plotLimits, annotationFont *truetype.Font) chart.AnnotationSeries {
	annotationSeries := chart.AnnotationSeries{
		Annotations: []chart.Value2{
			{
				Label:  threshold.title,
				XValue: threshold.timePoint,
				YValue: threshold.value,
				Style: chart.Style{
					Show:        true,
					Padding:     chart.Box{Right: threshold.growTo},
					Font:        annotationFont,
					FontSize:    8,
					FontColor:   chart.ColorAlternateGray,
					StrokeColor: chart.ColorAlternateGray,
					FillColor:   drawing.ColorFromHex(threshold.color).WithAlpha(20),
				},
			},
		},
	}
	return annotationSeries
}
