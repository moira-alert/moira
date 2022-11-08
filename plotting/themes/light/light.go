package light

import (
	"github.com/beevee/go-chart"
	"github.com/beevee/go-chart/drawing"
	"github.com/golang/freetype/truetype"
)

// PlotTheme implements moira.PlotTheme interface
type PlotTheme struct {
	font              *truetype.Font
	fontSizePrimary   float64
	fontSizeSecondary float64
	bgColor           string
	curveColors       []string
}

// NewTheme returns light theme
func NewTheme(themeFont *truetype.Font) (*PlotTheme, error) {
	return &PlotTheme{
		font:              themeFont,
		fontSizePrimary:   10, //nolint
		fontSizeSecondary: 8,  //nolint
		bgColor:           `ffffff`,
		curveColors: []string{
			`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
			`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
		},
	}, nil
}

// GetTitleStyle returns title style
func (theme *PlotTheme) GetTitleStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    15, //nolint
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(theme.bgColor),
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

// GetGridStyle returns grid style
func (theme *PlotTheme) GetGridStyle() chart.Style {
	return chart.Style{
		Show:        true,
		StrokeColor: drawing.ColorFromHex(`1f1d1d`),
		StrokeWidth: 0.03, //nolint
	}
}

// GetCanvasStyle returns canvas style
func (theme *PlotTheme) GetCanvasStyle() chart.Style {
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
	}
}

// GetBackgroundStyle returns background style
func (theme *PlotTheme) GetBackgroundStyle(maxMarkLen int) chart.Style {
	verticalShift := 40
	horizontalShift := 20
	if maxMarkLen > 4 { //nolint
		horizontalShift = horizontalShift / 2 //nolint
	}
	return chart.Style{
		FillColor: drawing.ColorFromHex(theme.bgColor),
		Padding: chart.Box{
			Top:    verticalShift,
			Bottom: verticalShift,
			Left:   horizontalShift,
			Right:  horizontalShift + (maxMarkLen * 6),
		},
	}
}

// GetThresholdStyle returns threshold style
func (theme *PlotTheme) GetThresholdStyle(thresholdType string) chart.Style {
	var thresholdColor string
	switch thresholdType {
	case "ERROR":
		thresholdColor = `8b0000`
	case "WARN":
		thresholdColor = `cccc00`
	}
	return chart.Style{
		Show:        true,
		StrokeWidth: 1,
		StrokeColor: drawing.ColorFromHex(thresholdColor).WithAlpha(90), //nolint
		FillColor:   drawing.ColorFromHex(thresholdColor).WithAlpha(20), //nolint
	}
}

// GetAnnotationStyle returns annotation style
func (theme *PlotTheme) GetAnnotationStyle(thresholdType string) chart.Style {
	var rightBoxDimension int
	var annotationColor string
	switch thresholdType {
	case "ERROR":
		annotationColor = `8b0000`
	case "WARN":
		annotationColor = `cccc00`
		rightBoxDimension = 9
	}
	return chart.Style{
		Show:        true,
		Padding:     chart.Box{Right: rightBoxDimension},
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: chart.ColorAlternateGray,
		FillColor:   drawing.ColorFromHex(annotationColor).WithAlpha(20), //nolint
	}
}

// GetSerieStyles returns curve and single point styles
func (theme *PlotTheme) GetSerieStyles(curveInd int) (chart.Style, chart.Style) {
	var curveColor drawing.Color
	if curveInd >= len(theme.curveColors)-1 {
		curveColor = drawing.ColorFromHex(theme.curveColors[0])
	} else {
		curveColor = drawing.ColorFromHex(theme.curveColors[curveInd])
	}
	curveWidth := float64(1)
	curveStyle := chart.Style{
		Show:        true,
		StrokeWidth: curveWidth,
		StrokeColor: curveColor.WithAlpha(90), //nolint
		FillColor:   curveColor.WithAlpha(20), //nolint
	}
	pointStyle := chart.Style{
		Show:        true,
		StrokeWidth: chart.Disabled,
		DotWidth:    curveWidth / 2,           //nolint
		DotColor:    curveColor.WithAlpha(90), //nolint
	}
	return curveStyle, pointStyle
}

// GetLegendStyle returns legend style
func (theme *PlotTheme) GetLegendStyle() chart.Style {
	return chart.Style{
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		FillColor:   drawing.ColorTransparent,
		StrokeColor: drawing.ColorTransparent,
	}
}

// GetXAxisStyle returns x axis style
func (theme *PlotTheme) GetXAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    theme.fontSizeSecondary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}

// GetYAxisStyle returns y axis style
func (theme *PlotTheme) GetYAxisStyle() chart.Style {
	return chart.Style{
		Show:        true,
		Font:        theme.font,
		FontSize:    theme.fontSizePrimary,
		FontColor:   chart.ColorAlternateGray,
		StrokeColor: drawing.ColorFromHex(theme.bgColor),
	}
}
