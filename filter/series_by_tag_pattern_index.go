package filter

import (
	"github.com/moira-alert/moira"
)

// SeriesByTagPatternIndex helps to index the seriesByTag patterns and allows to match them by metric.
type SeriesByTagPatternIndex struct {
	// namesPrefixTree stores MatchingHandler's for patterns that have name tag in prefix tree structure
	namesPrefixTree *PrefixTree
	// withoutStrictNameTagPatternMatchers stores MatchingHandler's for patterns that have no name tag
	withoutStrictNameTagPatternMatchers map[string]MatchingHandler
	// Flags for compatibility with different graphite behaviours
	compatibility Compatibility
}

// NewSeriesByTagPatternIndex creates new SeriesByTagPatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag.
func NewSeriesByTagPatternIndex(
	logger moira.Logger,
	tagSpecsByPattern map[string][]TagSpec,
	compatibility Compatibility,
) *SeriesByTagPatternIndex {
	namesPrefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}
	withoutStrictNameTagPatternMatchers := make(map[string]MatchingHandler)

	for pattern, tagSpecs := range tagSpecsByPattern {
		nameTagValue, matchingHandler, err := CreateMatchingHandlerForPattern(tagSpecs, &compatibility)
		if err != nil {
			logger.Error().
				Error(err).
				String("pattern", pattern).
				Msg("Failed to create MatchingHandler for pattern")
			continue
		}

		if nameTagValue == "" {
			withoutStrictNameTagPatternMatchers[pattern] = matchingHandler
		} else {
			namesPrefixTree.AddWithPayload(nameTagValue, pattern, matchingHandler)
		}
	}

	return &SeriesByTagPatternIndex{
		compatibility:                       compatibility,
		namesPrefixTree:                     namesPrefixTree,
		withoutStrictNameTagPatternMatchers: withoutStrictNameTagPatternMatchers,
	}
}

// MatchPatterns allows to match patterns by metric name and its labels.
func (index *SeriesByTagPatternIndex) MatchPatterns(metricName string, labels map[string]string) []string {
	matchedPatterns := make([]string, 0)

	matchingHandlersWithCorrespondingNameTag := index.namesPrefixTree.MatchWithValue(metricName)
	for pattern, matchingHandler := range matchingHandlersWithCorrespondingNameTag {
		if matchingHandler(metricName, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}

	for pattern, matchingHandler := range index.withoutStrictNameTagPatternMatchers {
		if matchingHandler(metricName, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}

	return matchedPatterns
}
