package plotting

import (
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const ThresholdSerie = "threshold" // ThresholdSerie is a name that indicates threshold

// Threshold represents threshold parameters
type Threshold struct {
	Title  string
	Value  float64
	Point  float64
	Color  string
	GrowTo int
}

// GenerateThresholds returns thresholds available for plot
func GenerateThresholds(plot Plot, limits Limits) []Threshold {
	var thresholds = make([]Threshold, 0)
	timePoint := float64(limits.To.UnixNano())
	if plot.ErrorValue != nil && limits.FormsSetContaining(*plot.ErrorValue) {
		thresholds = append(thresholds, Threshold{
			Title:  "ERROR",
			Value:  *plot.ErrorValue,
			Point:  timePoint,
			Color:  ErrorThreshold,
			GrowTo: 0,
		})
	}
	if plot.WarnValue != nil && limits.FormsSetContaining(*plot.WarnValue) {
		if plot.ErrorValue == nil || *plot.WarnValue != *plot.ErrorValue {
			thresholds = append(thresholds, Threshold{
				Title:  "WARN",
				Value:  *plot.WarnValue,
				Point:  timePoint,
				Color:  WarningThreshold,
				GrowTo: 9,
			})
		}
	}
	return thresholds
}

// GenerateThresholdSeries returns threshold series
func (threshold Threshold) GenerateThresholdSeries(limits Limits, isRaising bool) chart.TimeSeries {
	thresholdValue := threshold.Value
	if isRaising {
		thresholdValue = limits.Highest - threshold.Value
	}
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
		thresholdSeries.YValues = append(thresholdSeries.YValues, thresholdValue)
	}
	return thresholdSeries
}

// GenerateAnnotationSeries returns threshold annotation series
func (threshold Threshold) GenerateAnnotationSeries(limits Limits, isRaising bool, annotationFont *truetype.Font) chart.AnnotationSeries {
	annotationValue := threshold.Value
	if isRaising {
		annotationValue = limits.Highest - threshold.Value
	}
	annotationSeries := chart.AnnotationSeries{
		Annotations: []chart.Value2{
			{
				Label:  threshold.Title,
				XValue: threshold.Point,
				YValue: annotationValue,
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
