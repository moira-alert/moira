package filter

import (
	"github.com/moira-alert/moira"
)

// PatternIndex helps to index patterns and allows to match them by metric
type PatternIndex struct {
	Tree *PrefixTree
}

// NewPatternIndex creates new PatternIndex using patterns
func NewPatternIndex(logger moira.Logger, patterns []string) *PatternIndex {
	prefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}
	for _, pattern := range patterns {
		prefixTree.Add(pattern)
	}

	return &PatternIndex{Tree: prefixTree}
}

// MatchPatterns allows matching pattern by metric
func (source *PatternIndex) MatchPatterns(metric string) []string {
	return source.Tree.Match(metric)
}
