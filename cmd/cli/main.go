package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
)

// Moira version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

var moiraValidVersions = []string{"2.2", "2.3"}

var (
	configFileName         = flag.String("config", "/etc/moira/cli.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

var (
	update    = flag.Bool("update", false, fmt.Sprintf("convert existing database structures into required ones for current Moira version. ou must choose required version using flag '-from-version'. Valid update versions is %s", strings.Join(moiraValidVersions, ", ")))
	downgrade = flag.Bool("downgrade", false, fmt.Sprintf("convert existing database structures into required ones for previous Moira version. You must choose required version using flag '-to-version'. Valid downgrade versions is %s", strings.Join(moiraValidVersions, ", ")))
)

var (
	updateFromVersion  = flag.String("from-version", "", "determines Moira version, FROM which need to UPDATE database structures.")
	downgradeToVersion = flag.String("to-version", "", "determines Moira version, TO which need to DOWNGRADE database structures")
)

const (
	stateErrorTag           = "ERROR"
	stateDegradationTag     = "DEGRADATION"
	stateHighDegradationTag = "HIGH DEGRADATION"
)

func main() {
	logger, dataBase := initApp()

	if *update {
		fromVersion := checkValidVersion(updateFromVersion, true)
		switch fromVersion {
		case "2.2":
			err := updateFrom22(logger, dataBase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to update from version %s: %s", fromVersion, err.Error())
				os.Exit(1)
			}
			fallthrough
		case "2.3":
			err := updateFrom23(logger, dataBase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to update from version %s: %s", fromVersion, err.Error())
				os.Exit(1)
			}
		}
	}

	if *downgrade {
		toVersion := checkValidVersion(downgradeToVersion, false)
		switch toVersion {
		case "2.2":
			err := downgradeTo23(logger, dataBase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to downgrade to version %s: %s", toVersion, err.Error())
				os.Exit(1)
			}
			err = downgradeTo22(logger, dataBase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to downgrade to version %s: %s", toVersion, err.Error())
				os.Exit(1)
			}
		case "2.3":
			err := downgradeTo23(logger, dataBase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to update to version %s: %s", toVersion, err.Error())
				os.Exit(1)
			}
		}
	}
}

func initApp() (moira.Logger, moira.Database) {
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

	logger, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	dataBase := redis.NewDatabase(logger, databaseSettings)
	return logger, dataBase
}

func checkValidVersion(updateFromVersion *string, isUpdate bool) string {
	validFlag := "-from-version"
	if !isUpdate {
		validFlag = "-to-version"
	}

	if updateFromVersion == nil || *updateFromVersion == "" || contains(moiraValidVersions, *updateFromVersion) {
		fmt.Fprintf(os.Stderr, "You must set valid '%s' flag. Valid versions is %s", validFlag, strings.Join(moiraValidVersions, ", "))
		os.Exit(1)
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
