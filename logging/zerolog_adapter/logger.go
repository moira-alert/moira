package logging

import (
	"fmt"
	"github.com/moira-alert/moira"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path/filepath"
)

type Logger struct {
	zerolog.Logger
}

const (
	ModuleFieldName   = "module"
	DefaultTimeFormat = "2006-01-02 15:04:05.000"
)

// ConfigureLog creates new logger based on github.com/rs/zerolog package
func ConfigureLog(logFile, logLevel, module string, pretty bool) (*Logger, error) {
	return newLog(logFile, logLevel, module, pretty, false)
}

// GetLogger need only for backward compatibility in tests
func GetLogger(module string) (moira.Logger, error) {
	return newLog("stdout", "info", module, true, true)
}

func newLog(logFile, logLevel, module string, pretty, colorOff bool) (*Logger, error) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}

	logWriter, err := getLogWriter(logFile)
	if err != nil {
		return nil, err
	}
	zerolog.TimeFieldFormat = DefaultTimeFormat

	if pretty {
		logWriter = zerolog.ConsoleWriter{
			Out:        logWriter,
			NoColor:    colorOff,
			TimeFormat: DefaultTimeFormat,
			PartsOrder: []string{zerolog.TimestampFieldName, ModuleFieldName, zerolog.LevelFieldName, zerolog.MessageFieldName},
		}
	}

	logger := zerolog.New(logWriter).Level(level).With().Str(ModuleFieldName, module).Logger()
	return &Logger{logger}, nil
}

func getLogWriter(logFileName string) (io.Writer, error) {
	if logFileName == "stdout" || logFileName == "" {
		return os.Stdout, nil
	}

	logDir := filepath.Dir(logFileName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("can't create log directories %s: %s", logDir, err.Error())
	}
	logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("can't open log file %s: %s", logFileName, err.Error())
	}
	return logFile, nil
}

func (l Logger) Debug(args ...interface{}) {
	l.Logger.Debug().Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug().Timestamp().Msgf(format, args...)
}

func (l Logger) Info(args ...interface{}) {
	l.Logger.Info().Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info().Timestamp().Msgf(format, args...)
}

func (l Logger) Error(args ...interface{}) {
	l.Logger.Error().Timestamp().Msgf(fmt.Sprint(args...))
}

func (l Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error().Timestamp().Msgf(format, args...)
}

func (l Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal().Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatal().Timestamp().Msgf(format, args...)
}

func (l Logger) Warning(args ...interface{}) {
	l.Logger.Warn().Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) Warningf(format string, args ...interface{}) {
	l.Logger.Warn().Timestamp().Msgf(format, args...)
}

func (l *Logger) String(key, value string) moira.Logger {
	l.Logger = l.Logger.With().Str(key, value).Logger()
	return l
}

func (l *Logger) Int(key string, value int) moira.Logger {
	l.Logger = l.Logger.With().Int(key, value).Logger()
	return l
}

func (l *Logger) Fields(fields map[string]interface{}) moira.Logger {
	l.Logger = l.Logger.With().Fields(fields).Logger()
	return l
}

func (l *Logger) Level(s string) (moira.Logger, error) {
	level, err := zerolog.ParseLevel(s)
	if err != nil {
		return l, err
	}
	l.Logger = l.Logger.Level(level)
	return l, nil
}
