package themes

import (
	"github.com/golang/freetype/truetype"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type darkTheme struct {
	font        *truetype.Font
	bgColor     string
	curveColors []string
}

func GetDarkTheme(themeFont *truetype.Font) (*darkTheme, error) {
	return &darkTheme{
		font: themeFont,
		bgColor: "1f1d1d",
		curveColors: []string{
			`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
			`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
		},
	}, nil
}

func (theme *darkTheme) GetTitleStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(theme.bgColor),
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *darkTheme) GetGridStyle() chart.Style {
	return chart.Style{
		Show:        true,
		StrokeColor: drawing.ColorFromHex("ffffff"),
		StrokeWidth: 0.03,
	}
}

func (theme *darkTheme) GetCanvasStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *darkTheme) GetBackgroundStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *darkTheme) GetThresholdStyle(thresholdType string) chart.Style {
	var thresholdColor string
	switch thresholdType {
	case "ERROR":
		thresholdColor = "ed2e18"
	case "WARN":
		thresholdColor = "f79520"
	}
	return chart.Style{
		Show:        true,
		StrokeWidth: 1,
		StrokeColor: drawing.ColorFromHex(thresholdColor).WithAlpha(90),
		FillColor:   drawing.ColorFromHex(thresholdColor).WithAlpha(20),
	}
}

func (theme *darkTheme) GetAnnotationStyle(thresholdType string) chart.Style {
	var rightBoxDimension int
	var annotationColor string
	switch thresholdType {
	case "ERROR":
		annotationColor = "ed2e18"
	case "WARN":
		annotationColor = "f79520"
		rightBoxDimension = 9
	}
	return chart.Style{
		Show:        true,
		Padding:     chart.Box{Right: rightBoxDimension},
		Font:        theme.font,
		FontSize:    8,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(annotationColor).WithAlpha(20),
	}
}

func (theme *darkTheme) GetCurveStyle(curveInd int) chart.Style {
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

func (theme *darkTheme) GetLegendStyle() chart.Style {
	return chart.Style{
		FontSize:    8.0,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorTransparent,
		StrokeColor: drawing.ColorTransparent,
	}
}

func (theme *darkTheme) GetXAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    8,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *darkTheme) GetYAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}
