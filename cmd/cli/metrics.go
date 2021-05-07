package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira"

	"github.com/spf13/viper"
)

func cleanupOutdatedMetrics(config cleanupMetricsConfig, database moira.Database, logger moira.Logger) error {
	duration, err := time.ParseDuration(config.HotParams.CleanupDuration)
	if err != nil {
		return err
	}

	batchCounter, totalCounter := 0, 0
	currentParams := config.HotParams
	keysBatch := make([]string, 0, currentParams.CleanupBatchCount)
	cursor := database.ScanMetricNames()

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
			if currentParams.CleanupBatchCount != newHotParams.CleanupBatchCount {
				logger.Info("Cleanup batch count was changed to: ", newHotParams.CleanupBatchCount)
			}
			currentParams = newHotParams
		}

		logger.Debug("Scan keys started")
		metricsKeys, err := cursor.Next()
		if err != nil {
			if !strings.Contains(err.Error(), "end") {
				logger.Error(err)
			}
			break
		}
		logger.Debugf("Found %d keys", len(metricsKeys))

		for _, metric := range metricsKeys {
			keysBatch = append(keysBatch, metric)
			batchCounter++
			if batchCounter >= currentParams.CleanupBatchCount {
				logger.Debugf("Cleanup batch: size %d, keys: %q", len(keysBatch), keysBatch)
				if err := flushBatch(database, keysBatch, duration, config.DebugMode, config.DryRunMode); err != nil {
					return err
				}
				totalCounter += batchCounter
				batchCounter = 0
				keysBatch = make([]string, 0, currentParams.CleanupBatchCount)
				logger.Infof("Total processed %d keys. Sleep between batches for %d seconds...", totalCounter,
					currentParams.CleanupBatchTimeoutSeconds)
				time.Sleep(time.Second * time.Duration(currentParams.CleanupBatchTimeoutSeconds))
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
		now := "1618560000" // for debug with dump only
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
	if err := viper.UnmarshalKey("cleanup_metrics.hot_params", &hp); err != nil {
		logger.Error("Failed to unmarshall config hot_params: ", err.Error())
		return cleanupMetricsHotParams{}, err
	}
	return hp, nil
}
