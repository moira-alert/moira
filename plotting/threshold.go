package plotting

import (
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

// Threshold represents threshold parameters
type Threshold struct {
	Title     string
	Value     float64
	TimePoint float64
	Color     string
	GrowTo    int
}

// GenerateThresholds returns thresholds available for plot
func GenerateThresholds(plot Plot, limits Limits, raising bool) []Threshold {
	// TODO: cover cases with negative warn & error values
	// TODO: cover cases with out-of-range thresholds (no annotations required)
	timePoint := float64(limits.To.UnixNano())
	thresholds := make([]Threshold, 0)
	switch plot.ErrorValue {
	case nil:
		if plot.WarnValue != nil && !(*plot.WarnValue < 0) {
			if limits.FormsSetContaining(*plot.WarnValue) {
				warnThreshold := Threshold{
					Title:     "WARN",
					Value:     *plot.WarnValue,
					TimePoint: timePoint,
					Color:     WarningThreshold,
					GrowTo:    9,
				}
				if raising {
					warnThreshold.Value = limits.Highest - *plot.WarnValue
				}
				thresholds = append(thresholds, warnThreshold)
			}
		}
	default:
		if !(*plot.ErrorValue < 0) {
			if limits.FormsSetContaining(*plot.ErrorValue) {
				errThreshold := Threshold{
					Title:     "ERROR",
					Value:     *plot.ErrorValue,
					TimePoint: timePoint,
					Color:     ErrorThreshold,
					GrowTo:    0,
				}
				if raising {
					errThreshold.Value = limits.Highest - *plot.ErrorValue
				}
				thresholds = append(thresholds, errThreshold)
			}
		}
		if plot.WarnValue != nil {
			deltaLimits := math.Abs(limits.Highest - limits.Lowest)
			deltaThresholds := math.Abs(*plot.ErrorValue - *plot.WarnValue)
			if !(*plot.WarnValue < 0) && deltaThresholds > (deltaLimits/InvertedThresholdGap) {
				if limits.FormsSetContaining(*plot.WarnValue) {
					warnThreshold := Threshold{
						Title:     "WARN",
						Value:     *plot.WarnValue,
						TimePoint: timePoint,
						Color:     WarningThreshold,
						GrowTo:    9,
					}
					if raising {
						warnThreshold.Value = limits.Highest - *plot.WarnValue
					}
					thresholds = append(thresholds, warnThreshold)
				}
			}
		}
	}
	return thresholds
}

// GenerateThresholdSeries returns threshold series
func (threshold Threshold) GenerateThresholdSeries(limits Limits, isRaising bool) chart.TimeSeries {
	thresholdSeries := chart.TimeSeries{
		Name: ThresholdSerie,
		Style: chart.Style{
			Show:        true,
			StrokeWidth: 1,
			StrokeColor: drawing.ColorFromHex(threshold.Color).WithAlpha(90),
			FillColor:   drawing.ColorFromHex(threshold.Color).WithAlpha(20),
		},

		XValues: []time.Time{limits.From, limits.To},
		YValues: []float64{},
	}
	for j := 0; j < len(thresholdSeries.XValues); j++ {
		thresholdSeries.YValues = append(thresholdSeries.YValues, threshold.Value)
	}
	return thresholdSeries
}

// GenerateAnnotationSeries returns threshold annotation series
func (threshold Threshold) GenerateAnnotationSeries(limits Limits, isRaising bool, annotationFont *truetype.Font) chart.AnnotationSeries {
	annotationSeries := chart.AnnotationSeries{
		Annotations: []chart.Value2{
			{
				Label:  threshold.Title,
				XValue: threshold.TimePoint,
				YValue: threshold.Value,
				Style: chart.Style{
					Show:        true,
					Padding:     chart.Box{Right: threshold.GrowTo},
					Font:        annotationFont,
					FontSize:    8,
					FontColor:   chart.ColorAlternateGray,
					StrokeColor: chart.ColorAlternateGray,
					FillColor:   drawing.ColorFromHex(threshold.Color).WithAlpha(20),
				},
			},
		},
	}
	return annotationSeries
}
