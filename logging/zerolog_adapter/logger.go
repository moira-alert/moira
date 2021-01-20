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
	moduleFieldName = "module"
)

// ConfigureLog creates new logger based on github.com/rs/zerolog package
func ConfigureLog(logFile, logLevel, module string, pretty bool) (*Logger, error) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}

	logWriter, err := getLogWriter(logFile)
	if err != nil {
		return nil, err
	}

	if pretty {
		logWriter = zerolog.ConsoleWriter{
			Out:        logWriter,
			NoColor:    false,
			TimeFormat: "2006-01-02 15:04:05.000",
			PartsOrder: []string{zerolog.TimestampFieldName, moduleFieldName, zerolog.LevelFieldName, zerolog.MessageFieldName},
		}
	}

	logger := zerolog.New(logWriter).Level(level).With().Str(moduleFieldName, module).Logger()
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
	l.Logger.Debug().Timestamp().Msg(fmt.Sprint(args))
}

func (l Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug().Timestamp().Msgf(format, args)
}

func (l Logger) Info(args ...interface{}) {
	l.Logger.Info().Timestamp().Msg(fmt.Sprint(args))
}

func (l Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info().Timestamp().Msgf(format, args)
}

func (l Logger) Error(args ...interface{}) {
	l.Logger.Error().Timestamp().Msgf(fmt.Sprint(args))
}

func (l Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error().Timestamp().Msgf(format, args)
}

func (l Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal().Timestamp().Msg(fmt.Sprint(args))
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatal().Timestamp().Msgf(format, args)
}

func (l Logger) Warning(args ...interface{}) {
	l.Logger.Warn().Timestamp().Msg(fmt.Sprint(args))
}

func (l Logger) Warningf(format string, args ...interface{}) {
	l.Logger.Warn().Timestamp().Msgf(format, args)
}

func (l Logger) Level(s string) (moira.Logger, error) {
	level, error := zerolog.ParseLevel(s)
	return Logger{l.Logger.Level(level)}, error
}

func (l Logger) DebugEvent() moira.LogEvent {
	return &LogEvent{l.Logger.Debug()}
}

func (l Logger) InfoEvent() moira.LogEvent {
	return &LogEvent{l.Logger.Info()}
}

func (l Logger) WarningEvent() moira.LogEvent {
	return &LogEvent{l.Logger.Warn()}
}

func (l Logger) ErrorEvent() moira.LogEvent {
	return &LogEvent{l.Logger.Error()}
}

func (l Logger) FatalEvent() moira.LogEvent {
	return &LogEvent{l.Logger.Fatal()}
}

type LogEvent struct {
	*zerolog.Event
}

func (l *LogEvent) String(key, value string) moira.LogEvent {
	l.Event = l.Event.Str(key, value)
	return l
}

func (l *LogEvent) Int(key string, value int) moira.LogEvent {
	l.Event = l.Event.Int(key, value)
	return l
}

func (l *LogEvent) Fields(fields map[string]interface{}) moira.LogEvent {
	l.Event = l.Event.Fields(fields)
	return l
}

func (l *LogEvent) Message(message string) {
	l.Event.Msg(message)
}

func (l *LogEvent) Messagef(format string, args ...interface{}) {
	l.Event.Msgf(format, args)
}
