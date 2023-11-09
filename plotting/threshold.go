package plotting

import (
	"time"

	"github.com/moira-alert/go-chart"
	"github.com/moira-alert/moira"
)

const (
	// thresholdSerie is a name that indicates threshold
	thresholdSerie = "threshold"
	//// thresholdGapCoefficient is max allowed area
	//// between thresholds as percentage of limits delta
	// thresholdGapCoefficient = 0.25
)

// threshold represents threshold parameters
type threshold struct {
	thresholdType string
	yCoordinate   float64
}

// newThreshold returns described threshold item
func newThreshold(triggerType, thresholdType string, thresholdValue, higherLimit float64) *threshold {
	var yCoordinate float64
	if triggerType == moira.RisingTrigger {
		yCoordinate = higherLimit - thresholdValue
	} else {
		yCoordinate = thresholdValue
	}
	return &threshold{
		thresholdType: thresholdType,
		yCoordinate:   yCoordinate,
	}
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
		// TODO: uncomment to use annotations if necessary, remove otherwise
		// thresholdSeriesList = append(thresholdSeriesList, plotThreshold.generateAnnotationSeries(theme, limits))
	}
	return thresholdSeriesList
}

// generateThresholds returns thresholds available for plot
func generateThresholds(trigger *moira.Trigger, limits plotLimits) []*threshold {
	thresholds := make([]*threshold, 0)
	// No thresholds required
	if trigger.WarnValue == nil && trigger.ErrorValue == nil {
		return thresholds
	}
	// Trigger has ERROR value and threshold can be drawn
	if trigger.ErrorValue != nil && limits.formsSetContaining(*trigger.ErrorValue) {
		thresholds = append(thresholds, newThreshold(
			trigger.TriggerType, "ERROR", *trigger.ErrorValue, limits.highest))
	}
	// Trigger has WARN value and threshold can be drawn when:
	if trigger.WarnValue != nil && limits.formsSetContaining(*trigger.WarnValue) {
		thresholds = append(thresholds, newThreshold(
			trigger.TriggerType, "WARN", *trigger.WarnValue, limits.highest))
	}
	/**
	// Trigger has ERROR value and threshold can be drawn
	errThresholdRequied := trigger.ErrorValue != nil && limits.formsSetContaining(*trigger.ErrorValue)
	if errThresholdRequied {
		thresholds = append(thresholds, newThreshold(
			trigger.TriggerType, "ERROR", *trigger.ErrorValue, limits.highest))
	}
	// Trigger has WARN value and threshold can be drawn when:
	warnThresholdRequired := trigger.WarnValue != nil && limits.formsSetContaining(*trigger.WarnValue)
	if warnThresholdRequired {
		if errThresholdRequied {
			deltaLimits := math.Abs(limits.highest - limits.lowest)
			deltaThresholds := math.Abs(*trigger.ErrorValue - *trigger.WarnValue)
			if deltaThresholds > thresholdGapCoefficient*deltaLimits {
				//// there is enough place to draw both of ERROR and WARN thresholds
				thresholds = append(thresholds, newThreshold(
					trigger.TriggerType, "WARN", *trigger.WarnValue, limits.highest))
			}
		} else {
			//// there is no ERROR threshold required
			thresholds = append(thresholds, newThreshold(
				trigger.TriggerType, "WARN", *trigger.WarnValue, limits.highest))
		}
	}
	*/
	return thresholds
}

// generateThresholdSeries returns threshold series
func (threshold *threshold) generateThresholdSeries(theme moira.PlotTheme, limits plotLimits) chart.TimeSeries {
	thresholdSeries := chart.TimeSeries{
		Name:    thresholdSerie,
		Style:   theme.GetThresholdStyle(threshold.thresholdType),
		XValues: []time.Time{limits.from, limits.to},
		YValues: []float64{},
	}
	for j := 0; j < len(thresholdSeries.XValues); j++ {
		thresholdSeries.YValues = append(thresholdSeries.YValues, threshold.yCoordinate)
	}
	return thresholdSeries
}

/**
// generateAnnotationSeries returns threshold annotation series
func (threshold *threshold) generateAnnotationSeries(theme moira.PlotTheme, limits plotLimits) chart.AnnotationSeries {
	annotationSeries := chart.AnnotationSeries{
		Annotations: []chart.Value2{
			{
				Label:  threshold.thresholdType,
				XValue: float64(limits.to.UnixNano()),
				YValue: threshold.yCoordinate,
				Style:  theme.GetAnnotationStyle(threshold.thresholdType),
			},
		},
	}
	return annotationSeries
}
*/
