package filter

import (
	"github.com/moira-alert/moira"
)

// PatternIndex helps to index patterns and allows to match them by metric.
type PatternIndex struct {
	Tree          *PrefixTree
	compatibility Compatibility
}

// NewPatternIndex creates new PatternIndex using patterns.
func NewPatternIndex(logger moira.Logger, patterns []string, compatibility Compatibility) *PatternIndex {
	prefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}
	for _, pattern := range patterns {
		prefixTree.Add(pattern)
	}

	return &PatternIndex{Tree: prefixTree, compatibility: compatibility}
}

// MatchPatterns allows matching pattern by metric.
func (source *PatternIndex) MatchPatterns(metric string) []string {
	return source.Tree.Match(metric)
}
