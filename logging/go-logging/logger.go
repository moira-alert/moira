package logging

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/logging"
	goLogging "github.com/op/go-logging"
	"os"
	"path/filepath"
)

func ConfigureLog(config *logging.Config) (moira.Logger, error) {
	log, err := goLogging.GetLogger("cache")
	if err != nil {
		return nil, fmt.Errorf("Can't initialize logger: %s", err.Error())
	}
	logLevel, err := goLogging.LogLevel(config.LogLevel)
	if err != nil {
		logLevel = goLogging.DEBUG
	}

	goLogging.SetFormatter(goLogging.MustStringFormatter("%{time:2006-01-02 15:04:05}\t%{level}\t%{message}"))
	logBackend, err := getLogBackend(config.LogFile)
	if err != nil {
		return nil, err
	}
	logBackend.Color = config.LogColor
	goLogging.SetBackend(logBackend)
	goLogging.SetLevel(logLevel, "cache")
	return log, nil
}

func getLogBackend(logFileName string) (*goLogging.LogBackend, error) {
	if logFileName == "stdout" || logFileName == "" {
		return goLogging.NewLogBackend(os.Stdout, "", 0), nil
	} else {
		logDir := filepath.Dir(logFileName)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("Can't create log directories %s: %s", logDir, err.Error())
		}
		logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("Can't open log file %s: %s", logFileName, err.Error())
		}
		return goLogging.NewLogBackend(logFile, "", 0), nil
	}
}
