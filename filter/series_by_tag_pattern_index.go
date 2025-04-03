package filter

import (
	lrucache "github.com/hashicorp/golang-lru/v2"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// SeriesByTagPatternIndex helps to index the seriesByTag patterns and allows to match them by metric.
type SeriesByTagPatternIndex struct {
	// namesPrefixTree stores MatchingHandler's for patterns that have name tag in prefix tree structure
	namesPrefixTree *PrefixTree
	// withoutStrictNameTagPatternMatchers stores MatchingHandler's for patterns that have no name tag
	withoutStrictNameTagPatternMatchers []patternAndHandler
	// Flags for compatibility with different graphite behaviours
	compatibility Compatibility
}

type patternAndHandler struct {
	pattern string
	handler MatchingHandler
}

// NewSeriesByTagPatternIndex creates new SeriesByTagPatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag.
func NewSeriesByTagPatternIndex(
	logger moira.Logger,
	tagSpecsByPattern map[string][]TagSpec,
	compatibility Compatibility,
	patternMatchingCache *lrucache.Cache[string, *patternMatchingCacheItem],
	metrics *metrics.FilterMetrics,
) *SeriesByTagPatternIndex {
	namesPrefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}
	withoutStrictNameTagPatternMatchers := make([]patternAndHandler, 0)

	var patternMatchingEvicted int64

	for pattern, tagSpecs := range tagSpecsByPattern {
		var patternMatching *patternMatchingCacheItem

		patternMatching, ok := patternMatchingCache.Get(pattern)
		if !ok {
			nameTagValue, matchingHandler, err := CreateMatchingHandlerForPattern(tagSpecs, &compatibility)
			if err != nil {
				logger.Error().
					Error(err).
					String("pattern", pattern).
					Msg("Failed to create MatchingHandler for pattern")

				continue
			}

			patternMatching = &patternMatchingCacheItem{
				nameTagValue:    nameTagValue,
				matchingHandler: matchingHandler,
			}

			if evicted := patternMatchingCache.Add(pattern, patternMatching); evicted {
				patternMatchingEvicted++
			}
		}

		if patternMatching.nameTagValue == "" {
			withoutStrictNameTagPatternMatchers = append(
				withoutStrictNameTagPatternMatchers,
				patternAndHandler{
					pattern: pattern,
					handler: patternMatching.matchingHandler,
				},
			)
		} else {
			namesPrefixTree.AddWithPayload(patternMatching.nameTagValue, pattern, patternMatching.matchingHandler)
		}
	}

	metrics.MarkPatternMatchingEvicted(patternMatchingEvicted)

	return &SeriesByTagPatternIndex{
		compatibility:                       compatibility,
		namesPrefixTree:                     namesPrefixTree,
		withoutStrictNameTagPatternMatchers: withoutStrictNameTagPatternMatchers,
	}
}

// MatchPatterns allows to match patterns by metric name and its labels.
func (index *SeriesByTagPatternIndex) MatchPatterns(metricName string, labels map[string]string) []string {
	matchedPatterns := make([]string, 0)

	callback := func(pattern string, matchingHandler MatchingHandler) {
		if matchingHandler(metricName, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}
	index.namesPrefixTree.MatchWithValue(metricName, callback)

	for _, patternAndHandler := range index.withoutStrictNameTagPatternMatchers {
		if patternAndHandler.handler(metricName, labels) {
			matchedPatterns = append(matchedPatterns, patternAndHandler.pattern)
		}
	}

	return matchedPatterns
}
