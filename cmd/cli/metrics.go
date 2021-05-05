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
	cursor.SetCountLimit(config.HotParams.CleanupKeyScanBatchCount)

	for {
		if newHotParams, err := getConfigHotParams(logger); err == nil {
			logger.Debugf("Hot params: %v", newHotParams)
			if currentParams.CleanupDuration != newHotParams.CleanupDuration {
				logger.Infof("Cleanup duration was changed '%s' to '%s', process will be restarted",
					currentParams.CleanupDuration, newHotParams.CleanupDuration)
				totalCounter = 0
				if err := cursor.Free(); err != nil {
					logger.Warning(err)
				}
				cursor = database.ScanMetricNames()
			}
			if currentParams.CleanupKeyScanBatchCount != newHotParams.CleanupKeyScanBatchCount {
				logger.Infof("Cleanup key scan count was changed '%s' to '%s'", currentParams.CleanupKeyScanBatchCount,
					newHotParams.CleanupKeyScanBatchCount)
				cursor.SetCountLimit(newHotParams.CleanupKeyScanBatchCount)
			}
			currentParams = newHotParams
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
				if err := flushBatch(database, keysBatch, duration, config.DebugMode, config.DryRunMode); err != nil {
					return err
				}
				totalCounter += batchCounter
				batchCounter = 0
			}
		}
	}

	if batchCounter > 0 {
		if err := flushBatch(database, keysBatch, duration, config.DebugMode, config.DryRunMode); err != nil {
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

func flushBatch(database moira.Database, keysBatch []string, duration time.Duration, debugMode bool, dryRunMode bool) error {
	toTs := getTimestampWithCleanupDuration(debugMode, duration)
	if dryRunMode {
		return nil
	}
	if err := database.RemoveMetricsValues(keysBatch, toTs); err != nil {
		return err
	}
	return nil
}

func getTimestampWithCleanupDuration(debugMode bool, duration time.Duration) int64 {
	lastTs := time.Now().UTC()
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
