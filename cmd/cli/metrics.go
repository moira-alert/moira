package main

import (
	"strconv"
	"time"

	"github.com/moira-alert/moira"

	"github.com/spf13/viper"
)

func cleanupOutdatedMetrics(config cleanupMetricsConfig, database moira.Database, logger moira.Logger) error {
	duration, err := parseDuration(config.HotParams.CleanupDuration)
	if err != nil {
		return err
	}

	batchCounter, totalCounter := 0, 0
	keysBatch := make([]string, 0, config.HotParams.CleanupBatchCount)
	currentParams := config.HotParams
	cursor := database.ScanMetricNames()

	for {
		if hotParams, err := getConfigHotParams(logger); err == nil {
			logger.Debugf("Hot params: %v", hotParams)
			if currentParams.CleanupDuration != hotParams.CleanupDuration {
				logger.Infof("Cleanup duration was changed '%s' to '%s', process will be restarted",
					currentParams.CleanupDuration, hotParams.CleanupDuration)
			}
			currentParams = hotParams
		}

		logger.Debug("Scan keys started")
		metricsKeys, err := cursor.Next()
		if err != nil {
			logger.Error(err)
			break
		}
		logger.Info("keys: ", metricsKeys)

		logger.Debug("Cleanup was started")
		for _, metric := range metricsKeys {
			keysBatch = append(keysBatch, metric)
			// todo: add elapsed time metric
			batchCounter++
			if batchCounter >= currentParams.CleanupBatchCount {
				if err := flushBatch(database, keysBatch, duration, config.DebugMode); err != nil {
					return err
				}
				totalCounter += batchCounter
				batchCounter = 0
			}
		}
	}

	if batchCounter > 0 {
		if err := flushBatch(database, keysBatch, duration, config.DebugMode); err != nil {
			return err
		}
		totalCounter += batchCounter
	}
	logger.Infof("Cleanup was finished, %d metrics processed", totalCounter)
	return nil
}

func parseDuration(durationString string) (time.Duration, error) {
	return time.ParseDuration(durationString)
}

func flushBatch(database moira.Database, keysBatch []string, duration time.Duration, debugMode bool) error {
	toTs := getTimestampWithCleanupDuration(debugMode, duration)
	if err := database.RemoveMetricsValues(keysBatch, toTs); err != nil {
		return err
	}
	return nil
}

func getTimestampWithCleanupDuration(debugMode bool, duration time.Duration) int64 {
	lastTs := time.Now().UTC() // todo: check that eq moira writes
	if debugMode {
		now := "1618491240" // for debug with dump only
		i, err := strconv.ParseInt(now, 10, 64)
		if err != nil {
			panic(err)
		}
		lastTs = time.Unix(i, 0)
	}
	toTs := lastTs.Add(duration).Unix()
	return toTs
}

func getConfigHotParams(logger moira.Logger) (cleanupMetricsHotParams, error) {
	hp := cleanupMetricsHotParams{}
	if err := viper.UnmarshalKey("hot_params", &hp); err != nil {
		logger.Error("Failed to unmarshall config hot_params: ", err.Error())
		return cleanupMetricsHotParams{}, err
	}
	return hp, nil
}
