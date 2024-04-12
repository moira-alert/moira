package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/support"
	_ "go.uber.org/automaxprocs"
)

// Moira version.
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

var moiraValidVersions = []string{"2.3", "2.6", "2.7", "2.9"}

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

var plotting = flag.Bool("plotting", false, "enable images in all notifications")

var removeSubscriptions = flag.String("remove-subscriptions", "", "Remove given subscriptions separated by semicolons.")

var (
	cleanupUsers          = flag.Bool("cleanup-users", false, "Disable/delete contacts and subscriptions of missing users")
	cleanupLastChecks     = flag.Bool("cleanup-last-checks", false, "Delete abandoned triggers last checks.")
	cleanupTags           = flag.Bool("cleanup-tags", false, "Delete abandoned tags.")
	cleanupMetrics        = flag.Bool("cleanup-metrics", false, "Delete outdated metrics.")
	cleanupPatternMetrics = flag.Bool("cleanup-pattern-metrics", false, "Delete outdated pattern metrics.")
	cleanupRetentions     = flag.Bool("cleanup-retentions", false, "Delete abandoned retentions.")
	userDel               = flag.String("user-del", "", "Delete all contacts and subscriptions for a user")
	fromUser              = flag.String("from-user", "", "Transfer subscriptions and contacts from user.")
	toUser                = flag.String("to-user", "", "Transfer subscriptions and contacts to user.")
)

var (
	removeMetricsByPrefix = flag.String("remove-metrics-by-prefix", "", "Remove metrics by prefix (e.g. my.super.metric.")
	removeAllMetrics      = flag.Bool("remove-all-metrics", false, "Remove all metrics.")
)

var (
	pushTriggerDump = flag.Bool("push-trigger-dump", false, "Get trigger dump in JSON from stdin and save it to redis")
	triggerDumpFile = flag.String("trigger-dump-file", "", "File that holds trigger dump JSON from api method response")
)

var (
	removeTriggersStartWith       = flag.String("remove-triggers-start-with", "", "Remove triggers which have ID starting with string parameter")
	removeUnusedTriggersStartWith = flag.String("remove-unused-triggers-start-with", "", "Remove unused triggers which have ID starting with string parameter")
)

func main() { //nolint
	confCleanup, logger, database := initApp()

	if *update {
		fromVersion := checkValidVersion(logger, updateFromVersion, true)
		switch fromVersion {
		case "2.3":
			err := updateFrom23(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update from version 2.3")
			}
		case "2.6":
			err := updateFrom26(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update from version 2.6")
			}
		case "2.7":
			err := updateFrom27(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update from version 2.7")
			}
		case "2.9":
			err := updateFrom29(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update from version 2.9")
			}
		}
	}

	if *downgrade {
		toVersion := checkValidVersion(logger, downgradeToVersion, false)
		switch toVersion {
		case "2.3":
			err := downgradeTo23(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update to version 2.3")
			}
		case "2.6":
			err := downgradeTo26(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update to version 2.6")
			}
		case "2.7":
			err := downgradeTo27(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update to version 2.7")
			}
		case "2.9":
			err := downgradeTo29(logger, database)
			if err != nil {
				logger.Fatal().
					Error(err).
					Msg("Fail to update to version 2.9")
			}
		}
	}

	if *plotting {
		if err := enablePlottingInAllSubscriptions(logger, database); err != nil {
			logger.Error().
				Error(err).
				Msg("Failed to enable images in all notifications")
		}
	}

	if *fromUser != "" || *toUser != "" {
		if err := transferUserSubscriptionsAndContacts(database, *fromUser, *toUser); err != nil {
			logger.Error().
				Error(err).
				Msg("Failed to transfer user subscriptions and contacts")
		}
	}

	if *userDel != "" {
		if err := deleteUser(database, *userDel); err != nil {
			logger.Error().
				Error(err).
				Msg("Failed to delete user")
		}
	}

	if *removeMetricsByPrefix != "" {
		log := logger.String(moira.LogFieldNameContext, "cleanup")
		log.Info().
			String("prefix", *removeMetricsByPrefix).
			Msg("Removing metrics by prefix started")

		if err := handleRemoveMetricsByPrefix(database, *removeMetricsByPrefix); err != nil {
			log.Error().
				Error(err).
				Msg("Failed to remove metrics by prefix")
		}
		log.Info().
			String("prefix", *removeMetricsByPrefix).
			Msg("Removing metrics by prefix finished")
	}

	if *removeAllMetrics {
		log := logger.String(moira.LogFieldNameContext, "cleanup")
		log.Info().Msg("Removing all metrics started")
		if err := handleRemoveAllMetrics(database); err != nil {
			log.Error().
				Error(err).
				Msg("Failed to remove all metrics")
		}
		log.Info().Msg("Removing all metrics finished")
	}

	if *removeTriggersStartWith != "" {
		log := logger.String(moira.LogFieldNameContext, "remove-triggers-start-with")
		if err := handleRemoveTriggersStartWith(logger, database, *removeTriggersStartWith); err != nil {
			log.Error().
				Error(err).
				Msg("Failed to remove triggers by prefix")
		}
	}

	if *removeUnusedTriggersStartWith != "" {
		log := logger.String(moira.LogFieldNameContext, "remove-unused-triggers-start-with")
		if err := handleRemoveUnusedTriggersStartWith(logger, database, *removeUnusedTriggersStartWith); err != nil {
			log.Error().
				Error(err).
				Msg("Failed to remove unused triggers by prefix")
		}
	}

	if *cleanupUsers {
		log := logger.String(moira.LogFieldNameContext, "cleanup-users")

		log.Info().
			Interface("user_whitelist", confCleanup.Whitelist).
			Msg("Cleanup users started")

		if err := handleCleanup(logger, database, confCleanup); err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup")
		}

		log.Info().Msg("Cleanup users finished")
	}

	if *cleanupMetrics {
		log := logger.String(moira.LogFieldNameContext, "cleanup-metrics")

		log.Info().Msg("Cleanup of outdated metrics started")
		err := handleCleanUpOutdatedMetrics(confCleanup, database)
		if err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup outdated metrics")
		}

		log.Info().Msg("Cleanup of outdated metrics finished")
	}

	if *cleanupPatternMetrics {
		log := logger.String(moira.LogFieldNameContext, "cleanup-pattern-metrics")

		log.Info().Msg("Cleanup of outdated pattern metrics started")

		count, err := handleCleanUpOutdatedPatternMetrics(database)
		if err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup outdated pattern metrics")
		}

		log.Info().
			Int64("deleted_pattern_metrics", count).
			Msg("Cleaned up outdated pattern metrics")

		log.Info().Msg("Cleanup of outdated pattern metrics finished")
	}

	if *cleanupLastChecks {
		log := logger.String(moira.LogFieldNameContext, "cleanup-last-checks")

		log.Info().Msg("Cleanup abandoned triggers last checks started")
		err := handleCleanUpAbandonedTriggerLastCheck(database)
		if err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup abandoned triggers last checks")
		}

		log.Info().Msg("Cleanup abandoned triggers last checks finished")
	}

	if *cleanupTags {
		log := logger.String(moira.LogFieldNameContext, "cleanup-tags")
		log.Info().Msg("Cleanup abandoned tags started")
		count, err := handleCleanUpAbandonedTags(database)
		if err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup abandoned tags")
		}
		log.Info().
			Int("abandoned_tags_deleted", count).
			Msg("Cleanup abandoned tags finished")
	}

	if *cleanupRetentions {
		log := logger.String(moira.LogFieldNameContext, "cleanup-retentions")

		log.Info().Msg("Cleanup of abandoned retentions started")
		err := handleCleanUpAbandonedRetentions(database)
		if err != nil {
			log.Error().
				Error(err).
				Msg("Failed to cleanup abandoned retentions")
		}
		log.Info().Msg("Cleanup of abandoned retentions finished")
	}

	if *pushTriggerDump {
		logger.Info().Msg("Dump push started")
		f, err := openFile(*triggerDumpFile, os.O_RDONLY)
		if err != nil {
			logger.Fatal().
				Error(err).
				Msg("Failed to open triggerDumpFile")
		}
		defer closeFile(f, logger)

		dump := &dto.TriggerDump{}
		err = json.NewDecoder(f).Decode(dump)
		if err != nil {
			logger.Fatal().
				Error(err).
				Msg("cannot decode trigger dump")
		}

		logger.Info().Msg(GetDumpBriefInfo(dump))
		if err := support.HandlePushTrigger(logger, database, &dump.Trigger); err != nil {
			logger.Fatal().
				Error(err).
				Msg("Failed to handle push trigger")
		}
		if err := support.HandlePushTriggerMetrics(logger, database, dump.Trigger.ID, dump.Metrics); err != nil {
			logger.Fatal().
				Error(err).
				Msg("Failed to handle push trigger metrics")
		}
		if err := support.HandlePushTriggerLastCheck(
			logger,
			database,
			dump.Trigger.ID,
			&dump.LastCheck,
			dump.Trigger.ClusterKey(),
		); err != nil {
			logger.Fatal().
				Error(err).
				Msg("Failed to handle push trigger last check")
		}
		logger.Info().Msg("Dump was pushed")
	}

	if *removeSubscriptions != "" {
		logger.Info().Msg("Start deletion of subscriptions")
		subscriptionIDs := strings.Split(*removeSubscriptions, ";")
		deleted := 0

		logger.Debug().
			Interface("subscription_ids", subscriptionIDs).
			Msg("Remove subscription IDs")

		for _, subscriptionID := range subscriptionIDs {
			if err := database.RemoveSubscription(strings.TrimSpace(subscriptionID)); err != nil {
				logger.Error().
					Error(err).
					String("subscription_id", subscriptionID).
					Msg("Failed to remove subscription")
				continue
			}
			deleted++
		}

		logger.Info().
			Int("deleted", deleted).
			Msg("Deletion of subscriptions finished")
	}
}

func GetDumpBriefInfo(dump *dto.TriggerDump) string {
	return fmt.Sprintf("\nDump info:\n"+
		" - created: %s\n"+
		" - trigger.id: %s\n"+
		" - metrics count: %d\n"+
		" - last_succesfull_check: %d\n",
		dump.Created, dump.Trigger.ID, len(dump.Metrics), dump.LastCheck.LastSuccessfulCheckTimestamp)
}

func initApp() (cleanupConfig, moira.Logger, moira.Database) {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira - alerting system based on graphite or prometheus data")
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

	if err := cmd.ReadConfig(*configFileName, &config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't read settings: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "cli", config.LogPrettyFormat)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	dataBase := redis.NewDatabase(logger, databaseSettings, redis.NotificationHistoryConfig{}, redis.NotificationConfig{}, redis.Cli)
	return config.Cleanup, logger, dataBase
}

func checkValidVersion(logger moira.Logger, updateFromVersion *string, isUpdate bool) string {
	validFlag := "--from-version"
	if !isUpdate {
		validFlag = "--to-version"
	}

	if updateFromVersion == nil || *updateFromVersion == "" || !contains(moiraValidVersions, *updateFromVersion) {
		logger.Fatal().
			String("valid_version", strings.Join(moiraValidVersions, ", ")).
			String("flag", validFlag).
			String("your_version", *updateFromVersion).
			Msg("You must set valid flag")
	}
	return moira.UseString(updateFromVersion)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func openFile(filePath string, mode int) (*os.File, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file is not specified")
	}
	file, err := os.OpenFile(filePath, mode, 0o666) //nolint:gofumpt,gomnd
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	return file, nil
}

func closeFile(f *os.File, logger moira.Logger) {
	if f != nil {
		if err := f.Close(); err != nil {
			logger.Fatal().
				Error(err).
				Msg("Failed to close file")
		}
	}
}
