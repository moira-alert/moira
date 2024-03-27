package zerolog_adapter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging"
)

type Logger struct {
	logger zerolog.Logger
}

const (
	ModuleFieldName   = "module"
	DefaultTimeFormat = "2006-01-02 15:04:05.000"
)

// ConfigureLog creates new logger based on github.com/rs/zerolog package.
func ConfigureLog(logFile, logLevel, module string, pretty bool) (*Logger, error) {
	return newLog(logFile, logLevel, module, pretty, false)
}

// GetLogger need only for backward compatibility in tests.
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
	if err := os.MkdirAll(logDir, 0755); err != nil { //nolint:gofumpt,gomnd
		return nil, fmt.Errorf("can't create log directories %s: %s", logDir, err.Error())
	}
	logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644) //nolint:gofumpt,gomnd
	if err != nil {
		return nil, fmt.Errorf("can't open log file %s: %s", logFileName, err.Error())
	}
	return logFile, nil
}

func (l Logger) Debug() logging.EventBuilder {
	return EventBuilder{event: l.logger.Debug()}
}

func (l Logger) Info() logging.EventBuilder {
	return EventBuilder{event: l.logger.Info()}
}

func (l Logger) Error() logging.EventBuilder {
	return EventBuilder{event: l.logger.Error()}
}

func (l Logger) Fatal() logging.EventBuilder {
	return EventBuilder{event: l.logger.Fatal()}
}

func (l Logger) Warning() logging.EventBuilder {
	return EventBuilder{event: l.logger.Warn()}
}

func (l *Logger) String(key, value string) moira.Logger {
	l.logger = l.logger.With().Str(key, value).Logger()
	return l
}

func (l *Logger) Int(key string, value int) moira.Logger {
	l.logger = l.logger.With().Int(key, value).Logger()
	return l
}

func (l *Logger) Int64(key string, value int64) moira.Logger {
	l.logger = l.logger.With().Int64(key, value).Logger()
	return l
}

func (l *Logger) Fields(fields map[string]interface{}) moira.Logger {
	l.logger = l.logger.With().Fields(fields).Logger()
	return l
}

func (l *Logger) Level(s string) (moira.Logger, error) {
	level, err := zerolog.ParseLevel(s)
	if err != nil {
		return l, err
	}
	l.logger = l.logger.Level(level)
	return l, nil
}

func (l Logger) Clone() moira.Logger {
	return &Logger{
		logger: l.logger.With().Logger(),
	}
}
