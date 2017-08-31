package logging

import (
	"fmt"
	goLogging "github.com/op/go-logging"
	"os"
	"path/filepath"
)

// ConfigureLog creates new logger based on github.com/op/go-logging package
func ConfigureLog(logFile, logLevel, module string) (*goLogging.Logger, error) {
	log, err := goLogging.GetLogger(module)
	if err != nil {
		return nil, fmt.Errorf("Can't initialize logger: %s", err.Error())
	}
	level, err := goLogging.LogLevel(logLevel)
	if err != nil {
		level = goLogging.DEBUG
	}

	goLogging.SetFormatter(goLogging.MustStringFormatter("%{time:2006-01-02 15:04:05}\t%{module}\t%{level}\t%{message}"))
	logBackend, err := getLogBackend(logFile)
	if err != nil {
		return nil, err
	}
	logBackend.Color = false
	goLogging.SetBackend(logBackend)
	goLogging.SetLevel(level, module)
	return log, nil
}

func getLogBackend(logFileName string) (*goLogging.LogBackend, error) {
	if logFileName == "stdout" || logFileName == "" {
		return goLogging.NewLogBackend(os.Stdout, "", 0), nil
	}

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
