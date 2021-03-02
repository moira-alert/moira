package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/support"
)

// Moira version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

var moiraValidVersions = []string{"2.3"}

var (
	configFileName         = flag.String("config", "/etc/moira/cli.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

var (
	update    = flag.Bool("update", false, fmt.Sprintf("convert database to Moira current version. You must choose required version using flag '-from-version'. Valid update versions is %s", strings.Join(moiraValidVersions, ", ")))
	downgrade = flag.Bool("downgrade", false, fmt.Sprintf("convert database to Moira previous version. You must choose required version using flag '-to-version'. Valid downgrade versions is %s", strings.Join(moiraValidVersions, ", ")))
)

var (
	updateFromVersion  = flag.String("from-version", "", "determines Moira version, FROM which need to UPDATE database structures.")
	downgradeToVersion = flag.String("to-version", "", "determines Moira version, TO which need to DOWNGRADE database structures")
)

var (
	plotting = flag.Bool("plotting", false, "enable images in all notifications")
)

var (
	cleanup  = flag.Bool("cleanup", false, "Disable/delete contacts and subscriptions of missing users")
	userDel  = flag.String("user-del", "", "Delete all contacts and subscriptions for a user")
	fromUser = flag.String("from-user", "", "Transfer subscriptions and contacts from user.")
	toUser   = flag.String("to-user", "", "Transfer subscriptions and contacts to user.")
)

var (
	pullTrigger        = flag.String("pull-trigger", "", "Get trigger from redis and save it to file")
	pullTriggerMetrics = flag.String("pull-trigger-metrics", "", "Get trigger patterns and metrics from redis and save it to file")
	pushTrigger        = flag.Bool("push-trigger", false, "Get trigger in JSON from file and save it to redis")
	pushTriggerMetrics = flag.String("push-trigger-metrics", "", "Get trigger patterns and metrics in JSON from strdin and save it to redis")
	triggerFile        = flag.String("trigger-file", "", "File that holds trigger JSON")
	triggerMetricsFile = flag.String("trigger-metrics-file", "", "File that holds trigger metrics JSON")
)

func main() { //nolint
	confCleanup, logger, dataBase := initApp()

	if *update {
		fromVersion := checkValidVersion(logger, updateFromVersion, true)
		switch fromVersion {
		case "2.3":
			err := updateFrom23(logger, dataBase)
			if err != nil {
				logger.Fatalf("Fail to update from version %s: %s", fromVersion, err.Error())
			}
		}
	}

	if *downgrade {
		toVersion := checkValidVersion(logger, downgradeToVersion, false)
		switch toVersion {
		case "2.3":
			err := downgradeTo23(logger, dataBase)
			if err != nil {
				logger.Fatalf("Fail to update to version %s: %s", toVersion, err.Error())
			}
		}
	}

	if *plotting {
		if err := enablePlottingInAllSubscriptions(logger, dataBase); err != nil {
			logger.Errorf("Failed to enable images in all notifications")
		}
	}

	if *fromUser != "" || *toUser != "" {
		if err := transferUserSubscriptionsAndContacts(dataBase, *fromUser, *toUser); err != nil {
			logger.Error(err)
		}
	}

	if *userDel != "" {
		if err := deleteUser(dataBase, *userDel); err != nil {
			logger.Error(err)
		}
	}

	if *cleanup {
		logger.Debugf("User whitelist: %#v", confCleanup.Whitelist)
		if err := handleCleanup(logger, dataBase, confCleanup); err != nil {
			logger.Error(err)
		}
	}

	if *pullTrigger != "" {
		f, err := openFile(*triggerFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer f.Close()

		t, err := support.HandlePullTrigger(logger, dataBase, *pullTrigger)
		if err != nil {
			logger.Fatal(err)
		}
		if err := json.NewEncoder(f).Encode(t); err != nil {
			logger.Fatalf("cannot marshall trigger: %w", err)
		}
	}

	if *pullTriggerMetrics != "" {
		f, err := openFile(*triggerMetricsFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer f.Close()

		m, err := support.HandlePullTriggerMetrics(logger, dataBase, *pullTriggerMetrics)
		if err != nil {
			logger.Fatal(err)
		}
		if err := json.NewEncoder(f).Encode(m); err != nil {
			logger.Fatalf("cannot marshall trigger metrics: %w", err)
		}
	}

	if *pushTrigger {
		f, err := openFile(*triggerFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer f.Close()

		if err := support.HandlePushTrigger(logger, dataBase, f); err != nil {
			logger.Fatal(err)
		}
	}

	if *pushTriggerMetrics != "" {
		f, err := openFile(*triggerMetricsFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer f.Close()

		if err := support.HandlePushTriggerMetrics(logger, dataBase, *pushTriggerMetrics, f); err != nil {
			logger.Fatal(err)
		}
	}
}

func initApp() (cleanupConfig, moira.Logger, moira.Database) {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira - alerting system based on graphite data")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
		os.Exit(0)
	}

	config := getDefault()
	if *printDefaultConfigFlag {
		cmd.PrintConfig(config)
		os.Exit(0)
	}

	err := cmd.ReadConfig(*configFileName, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read settings: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "cli", config.LogPrettyFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	dataBase := redis.NewDatabase(logger, databaseSettings, redis.Cli)
	return config.Cleanup, logger, dataBase
}

func checkValidVersion(logger moira.Logger, updateFromVersion *string, isUpdate bool) string {
	validFlag := "--from-version"
	if !isUpdate {
		validFlag = "--to-version"
	}

	if updateFromVersion == nil || *updateFromVersion == "" || !contains(moiraValidVersions, *updateFromVersion) {
		logger.Fatalf("You must set valid '%s' flag. Valid versions is %s", validFlag, strings.Join(moiraValidVersions, ", "))
	}
	return *updateFromVersion
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func openFile(filePath string) (*os.File, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file is not specified")
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot create file: %w", err)
	}
	return file, nil
}
