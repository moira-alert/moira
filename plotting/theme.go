package plotting

import (
	"github.com/golang/freetype/truetype"
	"github.com/moira-alert/moira"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	darkTheme  = "dark"
	lightTheme = "light"
)

// PlotTheme is a structure to store theme parameters
type plotTheme struct {
	font                *truetype.Font
	bgColor             string
	warnThresholdColor  string
	errorThresholdColor string
	curveColors         []string
	gridStyle           chart.Style
}

func getPlotTheme(theme string) (*plotTheme, error) {
	themeFont, err := getDefaultFont()
	if err != nil {
		return nil, err
	}
	switch theme {
	case lightTheme:
		return &plotTheme{
			font:                themeFont,
			bgColor:             "ffffff",
			warnThresholdColor:  "f79520",
			errorThresholdColor: "ed2e18",
			curveColors: []string{
				`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
				`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
			},
			gridStyle: chart.Style{
				Show:        true,
				StrokeColor: drawing.ColorFromHex("1f1d1d"),
				StrokeWidth: 0.03,
			},
		}, nil
	case darkTheme:
		fallthrough
	default:
		return &plotTheme{
			font:                themeFont,
			bgColor:             "1f1d1d",
			warnThresholdColor:  "f79520",
			errorThresholdColor: "ed2e18",
			curveColors: []string{
				`89da59`, `90afc5`, `375e97`, `ffbb00`, `5bc8ac`, `4cb5f5`, `6ab187`, `ec96a4`,
				`f0810f`, `f9a603`, `a1be95`, `e2dfa2`, `ebdf00`, `5b7065`, `eb8a3e`, `217ca3`,
			},
			gridStyle: chart.Style{
				Show:        true,
				StrokeColor: drawing.ColorFromHex("ffffff"),
				StrokeWidth: 0.03,
			},
		}, nil
	}
}

func (theme *plotTheme) pickCurveColor(seriesInd int) string {
	if seriesInd >= len(theme.curveColors)-1 {
		return theme.curveColors[seriesInd]
	}
	return theme.curveColors[0]
}

// getDefaultFont returns true type font
func getDefaultFont() (*truetype.Font, error) {
	ttf, err := truetype.Parse(segoeUI)
	if err != nil {
		return nil, err
	}
	return ttf, nil
}

// getYAxisParams returns threshold specific params for yaxis
func getYAxisParams(triggerType string) (int, bool) {
	if triggerType == moira.RisingTrigger {
		return 1, true
	}
	return 0, false
}

// GetBgPadding returns background padding
func getBgPadding(plotLimits plotLimits, hasThresholds bool) chart.Box {
	// TODO: simplify this method
	if (plotLimits.highest - plotLimits.lowest) > 1000 {
		if hasThresholds {
			return chart.Box{Top: 40, Left: 15, Right: 21, Bottom: 40}
		}
		return chart.Box{Top: 40, Left: 15, Right: 65, Bottom: 40}
	}
	if hasThresholds {
		return chart.Box{Top: 40, Left: 30, Bottom: 40}
	}
	return chart.Box{Top: 40, Left: 30, Right: 49, Bottom: 40}
}
