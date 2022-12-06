package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/moira-alert/moira"
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
	event := l.Logger.Debug()
	if event == nil {
		return
	}
	event.Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) DebugWithError(msg string, err error) {
	l.Debugb().
		Error(err).
		Msg(msg)
}

func (l Logger) Debugf(format string, args ...interface{}) {
	event := l.Logger.Debug()
	if event == nil {
		return
	}
	event.Timestamp().Msgf(format, args...)
}

func (l Logger) Debugb() moira.EventBuilder {
	return EventBuilder{Event: l.Logger.Debug()}
}

func (l Logger) Info(args ...interface{}) {
	event := l.Logger.Info()
	if event == nil {
		return
	}
	event.Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) InfoWithError(msg string, err error) {
	l.Infob().
		Error(err).
		Msg(msg)
}

func (l Logger) Infof(format string, args ...interface{}) {
	event := l.Logger.Info()
	if event == nil {
		return
	}
	event.Timestamp().Msgf(format, args...)
}

func (l Logger) Infob() moira.EventBuilder {
	return EventBuilder{Event: l.Logger.Info()}
}

func (l Logger) Error(args ...interface{}) {
	event := l.Logger.Error()
	if event == nil {
		return
	}
	event.Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) ErrorWithError(msg string, err error) {
	l.Errorb().
		Error(err).
		Msg(msg)
}

func (l Logger) Errorf(format string, args ...interface{}) {
	event := l.Logger.Error()
	if event == nil {
		return
	}
	event.Timestamp().Msgf(format, args...)
}

func (l Logger) Errorb() moira.EventBuilder {
	return EventBuilder{Event: l.Logger.Error()}
}

func (l Logger) Fatal(args ...interface{}) {
	event := l.Logger.Fatal()
	if event == nil {
		return
	}
	event.Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) FatalWithError(msg string, err error) {
	l.Fatalb().
		Error(err).
		Msg(msg)
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	event := l.Logger.Fatal()
	if event == nil {
		return
	}
	event.Timestamp().Msgf(format, args...)
}

func (l Logger) Fatalb() moira.EventBuilder {
	return EventBuilder{Event: l.Logger.Fatal()}
}

func (l Logger) Warning(args ...interface{}) {
	event := l.Logger.Warn()
	if event == nil {
		return
	}
	event.Timestamp().Msg(fmt.Sprint(args...))
}

func (l Logger) WarningWithError(msg string, err error) {
	l.Warningb().
		Error(err).
		Msg(msg)
}

func (l Logger) Warningf(format string, args ...interface{}) {
	event := l.Logger.Warn()
	if event == nil {
		return
	}
	event.Timestamp().Msgf(format, args...)
}

func (l Logger) Warningb() moira.EventBuilder {
	return EventBuilder{Event: l.Logger.Warn()}
}

func (l *Logger) String(key, value string) moira.Logger {
	l.Logger = l.Logger.With().Str(key, value).Logger()
	return l
}

func (l *Logger) Int(key string, value int) moira.Logger {
	l.Logger = l.Logger.With().Int(key, value).Logger()
	return l
}

func (l *Logger) Int64(key string, value int64) moira.Logger {
	l.Logger = l.Logger.With().Int64(key, value).Logger()
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

func (l Logger) Clone() moira.Logger {
	return &Logger{
		Logger: l.Logger.With().Logger(),
	}
}
