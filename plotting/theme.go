package plotting

import (
	"github.com/golang/freetype/truetype"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/plotting/fonts"
	"github.com/moira-alert/moira/plotting/themes/grafana_dark"
	"github.com/moira-alert/moira/plotting/themes/grafana_light"
)

// getPlotTheme returns plot theme
func getPlotTheme(plotTheme string) (moira.PlotTheme, error) {
	// TODO: rewrite light theme
	var err error
	var theme moira.PlotTheme
	themeFont, err := getDefaultFont()
	if err != nil {
		return nil, err
	}
	switch plotTheme {
	case "light":
		theme, err = grafana_light.NewTheme(themeFont)
		if err != nil {
			return nil, err
		}
	case "dark":
		fallthrough
	default:
		theme, err = grafana_dark.NewTheme(themeFont)
		if err != nil {
			return nil, err
		}
	}
	return theme, nil
}

// getDefaultFont returns default font
func getDefaultFont() (*truetype.Font, error) {
	ttf, err := truetype.Parse(fonts.SegoeUI)
	if err != nil {
		return nil, err
	}
	return ttf, nil
}
