package themes

import (
	"github.com/golang/freetype/truetype"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type lightTheme struct {
	font        *truetype.Font
	bgColor     string
	curveColors []string
}

func GetLightTheme(themeFont *truetype.Font) (*lightTheme, error) {
	return &lightTheme{
		font: themeFont,
		bgColor: "ffffff",
		curveColors: []string{
			`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
			`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
		},
	}, nil
}

func (theme *lightTheme) GetTitleStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(theme.bgColor),
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *lightTheme) GetGridStyle() chart.Style {
	return chart.Style{
		Show:        true,
		StrokeColor: drawing.ColorFromHex("1f1d1d"),
		StrokeWidth: 0.03,
	}
}

func (theme *lightTheme) GetCanvasStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *lightTheme) GetBackgroundStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *lightTheme) GetThresholdStyle(thresholdType string) chart.Style {
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

func (theme *lightTheme) GetAnnotationStyle(thresholdType string) chart.Style {
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

func (theme *lightTheme) GetCurveStyle(curveInd int) chart.Style {
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

func (theme *lightTheme) GetLegendStyle() chart.Style {
	return chart.Style{
		FontSize:    8.0,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorTransparent,
		StrokeColor: drawing.ColorTransparent,
	}
}

func (theme *lightTheme) GetXAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    8,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

func (theme *lightTheme) GetYAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}
