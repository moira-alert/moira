package filter

import (
	"context"
	"fmt"
	"testing"
	"time"

	mock_clock "github.com/moira-alert/moira/mock/clock"
	"github.com/stretchr/testify/require"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"go.uber.org/mock/gomock"
)

func TestProcessIncomingMetric(t *testing.T) {
	testPatterns := []string{
		"cpu.used",
		"plain.metric",
		"seriesByTag(\"name=cpu.used\")",
		"seriesByTag(\"name=tag.metric\", \"tag1=val1\")",
		"seriesByTag(\"name=name.metric\")",
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)

	patternStorageCfg := PatternStorageConfig{
		PatternMatchingCacheSize: 100,
	}

	database.EXPECT().GetPatterns().Return(nil, fmt.Errorf("some error here"))

	metricRegistry, err := metrics.NewMetricContext(context.Background()).CreateRegistry()
	require.NoError(t, err)

	filterMetrics, err := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry(), metricRegistry)
	require.NoError(t, err)

	_, err = NewPatternStorage(patternStorageCfg, database, filterMetrics, logger, Compatibility{AllowRegexLooseStartMatch: true})
	require.EqualError(t, err, "failed to refresh pattern storage: some error here")

	database.EXPECT().GetPatterns().Return(testPatterns, nil)

	metricRegistry, err = metrics.NewMetricContext(context.Background()).CreateRegistry()
	require.NoError(t, err)

	filterMetrics, err = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry(), metricRegistry)
	require.NoError(t, err)

	patternsStorage, err := NewPatternStorage(
		patternStorageCfg,
		database,
		filterMetrics,
		logger,
		Compatibility{AllowRegexLooseStartMatch: true},
	)
	require.NoError(t, err)

	systemClock := mock_clock.NewMockClock(mockCtrl)
	systemClock.EXPECT().NowUTC().Return(time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC)).AnyTimes()
	patternsStorage.clock = systemClock

	t.Run("When invalid metric arrives, should be properly counted", func(t *testing.T) {
		matchedMetrics := patternsStorage.ProcessIncomingMetric(nil, time.Hour)
		require.Nil(t, matchedMetrics)
		require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
		require.Equal(t, int64(0), patternsStorage.metrics.ValidMetricsReceived.Count())
		require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
	})

	t.Run("When valid non-matching metric arrives", func(t *testing.T) {
		metricRegistry, err := metrics.NewMetricContext(context.Background()).CreateRegistry()
		require.NoError(t, err)

		patternsStorage.metrics, _ = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry(), metricRegistry)

		t.Run("For plain metric", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used 12 1234567890"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For tag metric", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used;tag1=val1 12 1234567890"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For plain metric which has the same pattern like name tag in tagged pattern", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("tag.metric 12 1234567890"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For tagged metric which body matches plain metric trigger pattern", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("plain.metric;tag1=val1 12 1234567890"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For plain metric which matches to tagged pattern which contains only name tag", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("name.metric 12 1234567890"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For too old metric should miss it", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used 12 123"), time.Hour)
			require.Nil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(0), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})
	})

	t.Run("When valid matching metric arrives", func(t *testing.T) {
		metricRegistry, err := metrics.NewMetricContext(context.Background()).CreateRegistry()
		require.NoError(t, err)

		patternsStorage.metrics, _ = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry(), metricRegistry)

		t.Run("For plain metric", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("plain.metric 12 1234567890"), time.Hour)
			require.NotNil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For tagged metric", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("tag.metric;tag1=val1 12 1234567890"), time.Hour)
			require.NotNil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})

		t.Run("For tagged metric which matches to tagged pattern which contains only name tag", func(t *testing.T) {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("name.metric;tag1=val1 12 1234567890"), time.Hour)
			require.NotNil(t, matchedMetrics)
			require.Equal(t, int64(1), patternsStorage.metrics.TotalMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.ValidMetricsReceived.Count())
			require.Equal(t, int64(1), patternsStorage.metrics.MatchingMetricsReceived.Count())
		})
	})

	t.Run("When ten valid metrics arrive match timer should be updated", func(t *testing.T) {
		metricRegistry, err := metrics.NewMetricContext(context.Background()).CreateRegistry()
		require.NoError(t, err)

		patternsStorage.metrics, _ = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry(), metricRegistry)
		for range 10 {
			patternsStorage.ProcessIncomingMetric([]byte("cpu.used 12 1234567890"), time.Hour)
		}

		require.Equal(t, int64(1), patternsStorage.metrics.MatchingTimer.Count())
	})

	mockCtrl.Finish()
}
