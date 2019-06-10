package filter

import (
	"fmt"
	"regexp"
)

var tagSpecRegex = regexp.MustCompile(`^["']([^,!=]+)\s*(!?=~?)\s*([^,]*)["']`)
var tagSpecDelimiterRegex = regexp.MustCompile(`^\s*,\s*`)
var seriesByTagRegex = regexp.MustCompile(`^seriesByTag\(([^)]+)\)$`)

// ErrNotSeriesByTag is returned if the pattern is not seriesByTag
var ErrNotSeriesByTag = fmt.Errorf("not seriesByTag pattern")

// TagSpecOperator represents an operator and it is used to query metric by tag value
type TagSpecOperator string

const (
	// EqualOperator is a strict equality operator and it is used to query metric by tag's value
	EqualOperator TagSpecOperator = "="
	// NotEqualOperator is a strict non-equality operator and it is used to query metric by tag's value
	NotEqualOperator TagSpecOperator = "!="
	// MatchOperator is a match operator which helps to match metric by regex
	MatchOperator TagSpecOperator = "=~"
	// NotMatchOperator is a non-match operator which helps not to match metric by regex
	NotMatchOperator TagSpecOperator = "!=~"
)

// TagSpec is a filter expression inside seriesByTag pattern
type TagSpec struct {
	Name     string
	Operator TagSpecOperator
	Value    string
}

// ParseSeriesByTag parses seriesByTag pattern and returns tags specs
func ParseSeriesByTag(input string) ([]TagSpec, error) {
	matchedSeriesByTagIndexes := seriesByTagRegex.FindStringSubmatchIndex(input)
	if len(matchedSeriesByTagIndexes) != 4 {
		return nil, ErrNotSeriesByTag
	}

	input = input[matchedSeriesByTagIndexes[2]:matchedSeriesByTagIndexes[3]]

	tagSpecs := make([]TagSpec, 0)

	for len(input) > 0 {
		if len(tagSpecs) > 0 {
			matchedTagSpecDelimiterIndexes := tagSpecDelimiterRegex.FindStringSubmatchIndex(input)
			if len(matchedTagSpecDelimiterIndexes) != 2 {
				return nil, ErrNotSeriesByTag
			}
			input = input[matchedTagSpecDelimiterIndexes[1]:]
		}

		matchedTagSpecIndexes := tagSpecRegex.FindStringSubmatchIndex(input)
		if len(matchedTagSpecIndexes) != 8 {
			return nil, ErrNotSeriesByTag
		}
		if input[matchedTagSpecIndexes[0]] != input[matchedTagSpecIndexes[1]-1] {
			return nil, ErrNotSeriesByTag
		}

		name := input[matchedTagSpecIndexes[2]:matchedTagSpecIndexes[3]]
		operator := TagSpecOperator(input[matchedTagSpecIndexes[4]:matchedTagSpecIndexes[5]])
		spec := input[matchedTagSpecIndexes[6]:matchedTagSpecIndexes[7]]

		tagSpecs = append(tagSpecs, TagSpec{name, operator, spec})

		input = input[matchedTagSpecIndexes[1]:]
	}
	return tagSpecs, nil
}

func createMatcher(spec TagSpec) func(string, map[string]string) bool {
	var matcherCondition func(string) bool
	allowMatchEmpty := false
	switch spec.Operator {
	case EqualOperator:
		allowMatchEmpty = true
		matcherCondition = func(value string) bool {
			return value == spec.Value
		}
	case NotEqualOperator:
		matcherCondition = func(value string) bool {
			return value != spec.Value
		}
	case MatchOperator:
		allowMatchEmpty = true
		matchRegex := regexp.MustCompile("^" + spec.Value)
		matcherCondition = func(value string) bool {
			return matchRegex.MatchString(value)
		}
	case NotMatchOperator:
		matchRegex := regexp.MustCompile("^" + spec.Value)
		matcherCondition = func(value string) bool {
			return !matchRegex.MatchString(value)
		}
	default:
		matcherCondition = func(_ string) bool {
			return false
		}
	}

	matchEmpty := matcherCondition("")
	return func(metric string, labels map[string]string) bool {
		if spec.Name == "name" {
			return matcherCondition(metric)
		}
		if value, found := labels[spec.Name]; found {
			return matcherCondition(value)
		}
		return allowMatchEmpty && matchEmpty
	}
}

// SeriesByTagPatternIndex helps to index the seriesByTag patterns and allows to match them by metric
type SeriesByTagPatternIndex struct {
	patternToMatcher map[string]func(string, map[string]string) bool
}

// NewSeriesByTagPatternIndex creates new SeriesByTagPatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag
func NewSeriesByTagPatternIndex(tagSpecsByPattern map[string][]TagSpec) *SeriesByTagPatternIndex {
	patternToMatcher := make(map[string]func(string, map[string]string) bool)
	for pattern, tagSpecs := range tagSpecsByPattern {
		matchers := make([]func(string, map[string]string) bool, 0)
		for _, tagSpec := range tagSpecs {
			matchers = append(matchers, createMatcher(tagSpec))
		}

		patternToMatcher[pattern] = func(metric string, labels map[string]string) bool {
			for _, matcher := range matchers {
				if !matcher(metric, labels) {
					return false
				}
			}
			return true
		}
	}

	return &SeriesByTagPatternIndex{patternToMatcher: patternToMatcher}
}

// MatchPatterns allows to match patterns by metric name and its labels
func (index *SeriesByTagPatternIndex) MatchPatterns(name string, labels map[string]string) []string {
	matchedPatterns := make([]string, 0)

	for pattern, matcher := range index.patternToMatcher {
		if matcher(name, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}

	return matchedPatterns
}
