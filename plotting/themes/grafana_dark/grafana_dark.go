package grafana_dark

import (
	"github.com/golang/freetype/truetype"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type plotTheme struct {
	font              *truetype.Font
	fontSizePrimary   float64
	fontSizeSecondary float64
	bgColor           string
	curveColors       []string
}

func NewTheme(themeFont *truetype.Font) (*plotTheme, error) {
	return &plotTheme{
		font:              themeFont,
		fontSizePrimary:   10,
		fontSizeSecondary: 8,
		bgColor:           `1f1d1d`,
		curveColors: []string{
			`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
			`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
		},
	}, nil
}

func (theme *plotTheme) GetTitleStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(theme.bgColor),
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *plotTheme) GetGridStyle() chart.Style {
	return chart.Style{
		Show:        true,
		StrokeColor: drawing.ColorFromHex(`ffffff`),
		StrokeWidth: 0.03,
	}
}

func (theme *plotTheme) GetCanvasStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *plotTheme) GetBackgroundStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
		Padding: chart.Box{
			Top:    40,
			Bottom: 40,
			Left:   2 * chart.DefaultYAxisMargin,
			Right:  2 + chart.DefaultYAxisMargin,
		},
	}
}

func (theme *plotTheme) GetThresholdStyle(thresholdType string) chart.Style {
	var thresholdColor string
	switch thresholdType {
	case "ERROR":
		thresholdColor = `ed2e18`
	case "WARN":
		thresholdColor = `f79520`
	}
	return chart.Style{
		Show:        true,
		StrokeWidth: 1,
		StrokeColor: drawing.ColorFromHex(thresholdColor).WithAlpha(90),
		FillColor:   drawing.ColorFromHex(thresholdColor).WithAlpha(20),
	}
}

func (theme *plotTheme) GetAnnotationStyle(thresholdType string) chart.Style {
	var rightBoxDimension int
	var annotationColor string
	switch thresholdType {
	case "ERROR":
		annotationColor = `ed2e18`
	case "WARN":
		annotationColor = `f79520`
		rightBoxDimension = 9
	}
	return chart.Style{
		Show:        true,
		Padding:     chart.Box{Right: rightBoxDimension},
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(annotationColor).WithAlpha(20),
	}
}

func (theme *plotTheme) GetCurveStyle(curveInd int) chart.Style {
	var curveColor string
	if curveInd >= len(theme.curveColors)-1 {
		curveColor = theme.curveColors[0]
	} else {
		curveColor = theme.curveColors[curveInd]
	}
	return chart.Style{
		Show:        true,
		StrokeWidth: 1,
		StrokeColor: drawing.ColorFromHex(curveColor).WithAlpha(90),
		FillColor:   drawing.ColorFromHex(curveColor).WithAlpha(20),
	}
}

func (theme *plotTheme) GetLegendStyle() chart.Style {
	return chart.Style{
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorTransparent,
		StrokeColor: drawing.ColorTransparent,
	}
}

func (theme *plotTheme) GetXAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *plotTheme) GetYAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    theme.fontSizePrimary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}
