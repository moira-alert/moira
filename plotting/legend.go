package plotting

import (
	"sort"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	deltaLabels      = 40
	deltaMarkerLabel = 26
	markerLength     = 10
	deltaMarker      = int(deltaLabels - markerLength)
	maxLabelsCount   = 4
	maxLabelLength   = 30
)

// GetPlotLegend returns plot legend
func GetPlotLegend(c *chart.Chart) chart.Renderable {
	// TODO: Simplify this method
	return func(r chart.Renderer, cb chart.Box, chartDefaults chart.Style) {
		legendDefault := chart.Style{
			FontSize:    8.0,
			FontColor:   chart.ColorAlternateGray,
			FillColor:   drawing.ColorTransparent,
			StrokeColor: drawing.ColorTransparent,
		}
		legendStyle := chartDefaults.InheritFrom(legendDefault)
		foundLabels := make(map[string]bool)

		labelsCount := 0
		var labels []string
		var lines []chart.Style
		for _, s := range c.Series {
			if s.GetStyle().IsZero() || s.GetStyle().Show {
				if _, isAnnotationSeries := s.(chart.AnnotationSeries); !isAnnotationSeries {
					legendLabel := s.GetName()
					_, isFound := foundLabels[legendLabel]
					if !isFound && legendLabel != ThresholdSerie {
						foundLabels[legendLabel] = true
						legendLabel := SanitizeLabelName(legendLabel, maxLabelLength)
						labels = append(labels, legendLabel)
						lines = append(lines, s.GetStyle())
						if labelsCount == maxLabelsCount-1 {
							break
						}
						labelsCount++
					}
				}
			}
		}

		sort.Sort(SortedByLen(labels))
		if len(labels) == maxLabelsCount {
			labels[len(labels)-1] = "other series"
			lines[len(lines)-1].StrokeColor = chart.ColorAlternateGray
		}

		legendStyle.GetTextOptions().WriteToRenderer(r)

		labelX := 0
		labelY := c.Height - 15
		markerY := labelY - int(legendStyle.FontSize/2)

		var label string
		for x := 0; x < len(labels); x++ {
			label = labels[x]
			if len(label) > 0 {
				textBoxForMeasure := r.MeasureText(label)
				itemXShiftForMeasure := textBoxForMeasure.Width() + deltaLabels
				labelX += itemXShiftForMeasure
			}
		}

		labelX = ((PlotWidth - (labelX - deltaLabels)) / 2) + (markerLength / 2)
		markerX := labelX + deltaMarkerLabel

		for x := 0; x < len(labels); x++ {
			label = labels[x]
			if len(label) > 0 {
				// Plotting markers
				r.SetStrokeColor(lines[x].GetStrokeColor())
				r.SetStrokeWidth(9)
				r.MoveTo(markerX-deltaLabels, markerY)
				r.LineTo(markerX-deltaMarker, markerY)
				r.Stroke()
				// Calculte marker and label shifts
				textBox := r.MeasureText(label)
				itemXShift := textBox.Width() + deltaLabels
				markerX += itemXShift
				// Plotting labels
				r.Text(label, labelX, labelY)
				labelX += itemXShift
			}
		}
	}
}
