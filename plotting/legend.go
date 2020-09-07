package plotting

import (
	"github.com/beevee/go-chart"
)

const (
	deltaLabels      = 40
	deltaMarkerLabel = 26
	markerLength     = 10
	deltaMarker      = int(deltaLabels - markerLength)
	maxLabelsCount   = 4
	maxLabelLength   = 30
)

type plotLine struct {
	label string
	style chart.Style
}

// getPlotLegend returns plot legend
func getPlotLegend(c *chart.Chart, legendStyle chart.Style, plotWidth int) chart.Renderable {
	// TODO: Simplify this method
	return func(r chart.Renderer, cb chart.Box, chartDefaults chart.Style) {
		foundLabels := make(map[string]bool)
		lines := make([]plotLine, 0, maxLabelsCount)

		for _, s := range c.Series {
			if s.GetStyle().IsZero() || s.GetStyle().Show {
				if _, isAnnotationSeries := s.(chart.AnnotationSeries); !isAnnotationSeries {
					legendLabel := s.GetName()
					_, isFound := foundLabels[legendLabel]
					if !isFound && legendLabel != thresholdSerie {
						foundLabels[legendLabel] = true

						legendLabel = sanitizeLabelName(legendLabel, maxLabelLength)
						lines = append(lines, plotLine{
							label: legendLabel,
							style: inheritFrom(s.GetStyle()),
						})

						if len(lines) == maxLabelsCount {
							break
						}
					}
				}
			}
		}

		if len(lines) == maxLabelsCount {
			lines[len(lines)-1].label = "other series"
			lines[len(lines)-1].style.StrokeColor = chart.ColorAlternateGray
		}

		legendStyle.GetTextOptions().WriteToRenderer(r)

		labelX := 0
		labelY := c.Height - 15 //nolint
		markerY := labelY - int(legendStyle.FontSize/2) //nolint

		for _, line := range lines {
			if len(line.label) > 0 {
				textBoxForMeasure := r.MeasureText(line.label)
				itemXShiftForMeasure := textBoxForMeasure.Width() + deltaLabels
				labelX += itemXShiftForMeasure
			}
		}

		labelX = ((plotWidth - (labelX - deltaLabels)) / 2) + (markerLength / 2)
		markerX := labelX + deltaMarkerLabel

		for _, line := range lines {
			if len(line.label) > 0 {
				// Plotting markers
				r.SetStrokeColor(line.style.GetStrokeColor())
				r.SetStrokeWidth(9) //nolint
				r.MoveTo(markerX-deltaLabels, markerY)
				r.LineTo(markerX-deltaMarker, markerY)
				r.Stroke()
				// Calculate marker and label shifts
				textBox := r.MeasureText(line.label)
				itemXShift := textBox.Width() + deltaLabels
				markerX += itemXShift
				// Plotting labels
				r.Text(line.label, labelX, labelY)
				labelX += itemXShift
			}
		}
	}
}

// inheritFrom inherits style from initial to make sure marker will be added
func inheritFrom(initial chart.Style) chart.Style {
	if initial.StrokeColor.IsZero() {
		initial.StrokeColor = initial.DotColor
		initial.StrokeWidth = 1
	}
	return initial
}
