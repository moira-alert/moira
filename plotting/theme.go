package plotting

import (
	"github.com/golang/freetype/truetype"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	// Classical Grafana-like dark theme
	DarkTheme = "1f1d1d"
	// Light theme
	LightTheme = "ffffff"
)

var (
	// Plot width
	PlotWidth = 800
	// Plot height
	PlotHeight = 400
	// Warning threshold color
	WarningThreshold = "f79520"
	// Error threshold color
	ErrorThreshold = "ed2e18"
	// Collection of Grafana-like colors
	CurveColors = []string{
		`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
		`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
	}
)

// GetDefaultFont returns true type font
func GetDefaultFont() (*truetype.Font, error) {
	ttf, err := truetype.Parse(SegoeUI)
	if err != nil {
		return nil, err
	}
	return ttf, nil
}

// GetGridStyle returns plot grid style
func GetGridStyle(plotTheme string) chart.Style {
	var styleColor string
	switch plotTheme {
	case DarkTheme:
		styleColor = "ffffff"
	case LightTheme:
		styleColor = "1f1d1d"
	}
	return chart.Style{
		Show:        true,
		StrokeColor: drawing.ColorFromHex(styleColor),
		StrokeWidth: 0.03,
	}
}

// GetYAxisParams returns threshold specific params for yaxis
func GetYAxisParams(isRaising bool) (int, bool) {
	if isRaising {
		return 1, true
	}
	return 0, false
}

// GetBgPadding returns background padding
func GetBgPadding(thresholdsCount int) chart.Box {
	if thresholdsCount > 0 {
		return chart.Box{Top: 40, Left: 30, Bottom: 40}
	}
	return chart.Box{Top: 40, Left: 20, Right: 40, Bottom: 40}
}
