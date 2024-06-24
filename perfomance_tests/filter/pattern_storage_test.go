// Package filter
// nolint
package filter

import (
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/filter"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func shufflePatterns(patterns []string) []string {
	rand.Shuffle(len(patterns), func(i, j int) {
		patterns[i], patterns[j] = patterns[j], patterns[i]
	})

	return patterns
}

func BenchmarkPatternStorageRefresh(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	filterMetrics := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
	logger, _ := logging.GetLogger("Benchmark")
	compatibility := filter.Compatibility{AllowRegexLooseStartMatch: true}
	database := mock_moira_alert.NewMockDatabase(mockCtrl)

	testcases := []struct {
		name      string
		cacheSize int
	}{
		{
			name:      "small_cache",
			cacheSize: 100,
		},
		{
			name:      "big_cache",
			cacheSize: 500,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		b.Run(testcase.name, func(b *testing.B) {
			plainPatterns, err := loadPatterns("plain_patterns.txt")
			if err != nil {
				b.Fatalf("failed to load plain patterns: %s", err.Error())
			}

			taggedPatterns, err := loadPatterns("tagged_patterns.txt")
			if err != nil {
				b.Fatalf("failed to load tagged patterns: %s", err.Error())
			}

			patterns := append(*plainPatterns, *taggedPatterns...)

			database.EXPECT().GetPatterns().Return(shufflePatterns(patterns), nil).AnyTimes()

			patternStorageCfg := filter.PatternStorageConfig{
				PatternMatchingCacheSize: testcase.cacheSize,
			}

			patternsStorage, err := filter.NewPatternStorage(patternStorageCfg, database, filterMetrics, logger, compatibility)
			if err != nil {
				logger.Fatal().Error(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := patternsStorage.Refresh(); err != nil {
					b.Fatalf("failed to refresh patterns storage: %s", err.Error())
				}
			}
		})
	}
}
