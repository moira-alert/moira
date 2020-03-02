package plotting

import (
	"github.com/golang/freetype/truetype"
	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/plotting/fonts"
	"github.com/moira-alert/moira/internal/plotting/themes/dark"
	"github.com/moira-alert/moira/internal/plotting/themes/light"
)

const (
	darkPlotTheme  = "dark"
	lightPlotTheme = "light"
)

// getPlotTheme returns plot theme
func getPlotTheme(plotTheme string) (moira2.PlotTheme, error) {
	// TODO: rewrite light theme
	var err error
	var theme moira2.PlotTheme
	themeFont, err := getDefaultFont()
	if err != nil {
		return nil, err
	}
	switch plotTheme {
	case darkPlotTheme:
		theme, err = dark.NewTheme(themeFont)
		if err != nil {
			return nil, err
		}
	case lightPlotTheme:
		fallthrough
	default:
		theme, err = light.NewTheme(themeFont)
		if err != nil {
			return nil, err
		}
	}
	return theme, nil
}

// getDefaultFont returns default font
func getDefaultFont() (*truetype.Font, error) {
	ttf, err := truetype.Parse(fonts.DejaVuSans)
	if err != nil {
		return nil, err
	}
	return ttf, nil
}
