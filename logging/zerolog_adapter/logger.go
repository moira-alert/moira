package logging

import (
	"fmt"
	"github.com/rs/zerolog"
)

type Logger struct {
	zerolog.Logger
}

func (l Logger) Debug(args ...interface{}) {
	l.Logger.Debug().Msg(fmt.Sprint(args))
}

func (l Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug().Msgf(format, args)
}

func (l Logger) Info(args ...interface{}) {
	l.Logger.Info().Msg(fmt.Sprint(args))
}

func (l Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info().Msgf(format, args)
}

func (l Logger) Error(args ...interface{}) {
	l.Logger.Error().Msgf(fmt.Sprint(args))
}

func (l Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error().Msgf(format, args)
}

func (l Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal().Msg(fmt.Sprint(args))
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatal().Msgf(format, args)
}

func (l Logger) Warning(args ...interface{}) {
	l.Logger.Warn().Msg(fmt.Sprint(args))
}

func (l Logger) Warningf(format string, args ...interface{}) {
	l.Logger.Warn().Msgf(format, args)
}
