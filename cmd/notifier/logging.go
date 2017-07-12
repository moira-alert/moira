package main

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/op/go-logging"
	"os"
	"path/filepath"
	"github.com/moira-alert/moira-alert/notifier"
)

func configureLog(config *notifier.Config) (moira.Logger, error) {
	var err error
	log, err := logging.GetLogger("notifier")
	if err != nil {
		return nil, fmt.Errorf("Can't initialize logger: %s", err.Error())
	}
	var logBackend *logging.LogBackend
	logLevel, err := logging.LogLevel(config.LogLevel)
	if err != nil {
		logLevel = logging.DEBUG
	}
	logging.SetFormatter(logging.MustStringFormatter("%{time:2006-01-02 15:04:05}\t%{level}\t%{message}"))
	logFileName := config.LogFile
	if logFileName == "stdout" || logFileName == "" {
		logBackend = logging.NewLogBackend(os.Stdout, "", 0)
	} else {
		logDir := filepath.Dir(logFileName)
		if err := os.MkdirAll(logDir, 755); err != nil {
			return nil, fmt.Errorf("Can't create log directories %s: %s", logDir, err.Error())
		}
		logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("Can't open log file %s: %s", logFileName, err.Error())
		}
		logBackend = logging.NewLogBackend(logFile, "", 0)
	}
	logBackend.Color = config.LogColor
	logging.SetBackend(logBackend)
	logging.SetLevel(logLevel, "notifier")
	return log, nil
}
