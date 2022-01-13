package filter

import (
	"fmt"
	"regexp"

	"github.com/moira-alert/moira"
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
	if len(matchedSeriesByTagIndexes) != 4 { //nolint
		return nil, ErrNotSeriesByTag
	}

	input = input[matchedSeriesByTagIndexes[2]:matchedSeriesByTagIndexes[3]]

	tagSpecs := make([]TagSpec, 0)

	for len(input) > 0 {
		if len(tagSpecs) > 0 {
			matchedTagSpecDelimiterIndexes := tagSpecDelimiterRegex.FindStringSubmatchIndex(input)
			if len(matchedTagSpecDelimiterIndexes) != 2 { //nolint
				return nil, ErrNotSeriesByTag
			}
			input = input[matchedTagSpecDelimiterIndexes[1]:]
		}

		matchedTagSpecIndexes := tagSpecRegex.FindStringSubmatchIndex(input)
		if len(matchedTagSpecIndexes) != 8 { //nolint
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

func createMatchingHandler(spec TagSpec) MatchingHandler {
	var matchingHandlerCondition func(string) bool
	allowMatchEmpty := false
	switch spec.Operator {
	case EqualOperator:
		allowMatchEmpty = true
		matchingHandlerCondition = func(value string) bool {
			return value == spec.Value
		}
	case NotEqualOperator:
		matchingHandlerCondition = func(value string) bool {
			return value != spec.Value
		}
	case MatchOperator:
		allowMatchEmpty = true
		matchRegex := regexp.MustCompile("^" + spec.Value)
		matchingHandlerCondition = func(value string) bool {
			return matchRegex.MatchString(value)
		}
	case NotMatchOperator:
		matchRegex := regexp.MustCompile("^" + spec.Value)
		matchingHandlerCondition = func(value string) bool {
			return !matchRegex.MatchString(value)
		}
	default:
		matchingHandlerCondition = func(_ string) bool {
			return false
		}
	}

	matchEmpty := matchingHandlerCondition("")
	return func(metric string, labels map[string]string) bool {
		if spec.Name == "name" {
			return matchingHandlerCondition(metric)
		}
		if value, found := labels[spec.Name]; found {
			return matchingHandlerCondition(value)
		}
		return allowMatchEmpty && matchEmpty
	}
}

type MatchingHandler func(string, map[string]string) bool

// SeriesByTagPatternIndex helps to index the seriesByTag patterns and allows to match them by metric
type SeriesByTagPatternIndex struct {
	// namesPatternIndex stores name tag values
	namesPatternIndex *PatternIndex
	// nameToPatternMatchers stores MatchingHandler's for name tag values
	nameToPatternMatchers map[string]map[string]MatchingHandler
	// withoutStrictNameTagPatternMatchers stores MatchingHandler's for patterns that have no name tag
	withoutStrictNameTagPatternMatchers map[string]MatchingHandler
}

// NewSeriesByTagPatternIndex creates new SeriesByTagPatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag
func NewSeriesByTagPatternIndex(logger moira.Logger, tagSpecsByPattern map[string][]TagSpec) *SeriesByTagPatternIndex {
	names := make(map[string]bool)
	nameToPatternMatchers := make(map[string]map[string]MatchingHandler)
	withoutStrictNameTagPatternMatchers := make(map[string]MatchingHandler)

	for pattern, tagSpecs := range tagSpecsByPattern {
		matchingHandlers := make([]MatchingHandler, 0)
		var nameTagValue string

		for _, tagSpec := range tagSpecs {
			if tagSpec.Name == "name" && tagSpec.Operator == EqualOperator {
				nameTagValue = tagSpec.Value
				names[nameTagValue] = true
			} else {
				matchingHandlers = append(matchingHandlers, createMatchingHandler(tagSpec))
			}
		}

		matchingHandler := func(metric string, labels map[string]string) bool {
			for _, matchingHandler := range matchingHandlers {
				if !matchingHandler(metric, labels) {
					return false
				}
			}
			return true
		}

		// Splitting all patterns into two maps: nameToPatternMatchers stores patterns with strict name tag and
		// withoutStrictNameTagPatternMatchers stores other patterns
		if nameTagValue == "" {
			withoutStrictNameTagPatternMatchers[pattern] = matchingHandler
		} else {
			if _, ok := nameToPatternMatchers[nameTagValue]; ok {
				nameToPatternMatchers[nameTagValue][pattern] = matchingHandler
			} else {
				nameToPatternMatchers[nameTagValue] = map[string]MatchingHandler{pattern: matchingHandler}
			}
		}
	}

	namePatternIndex := NewPatternIndex(logger, getMapKeys(names))
	return &SeriesByTagPatternIndex{
		namesPatternIndex:                   namePatternIndex,
		nameToPatternMatchers:               nameToPatternMatchers,
		withoutStrictNameTagPatternMatchers: withoutStrictNameTagPatternMatchers}
}

// MatchPatterns allows to match patterns by metric name and its labels
func (index *SeriesByTagPatternIndex) MatchPatterns(name string, labels map[string]string) []string {
	matchedPatterns := make([]string, 0)

	matchingHandlersWithCorrespondingNameTag := getMatchingHandlersWithCorrespondingNameTag(index, name)
	for pattern, matchingHandler := range matchingHandlersWithCorrespondingNameTag {
		if matchingHandler(name, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}

	for pattern, matchingHandler := range index.withoutStrictNameTagPatternMatchers {
		if (matchingHandler)(name, labels) {
			matchedPatterns = append(matchedPatterns, pattern)
		}
	}

	return matchedPatterns
}

func getMatchingHandlersWithCorrespondingNameTag(index *SeriesByTagPatternIndex, metricName string) map[string]MatchingHandler {
	patternsToMatchingHandlers := make(map[string]MatchingHandler)
	namesPatterns := index.namesPatternIndex.MatchPatterns(metricName)
	for _, patternName := range namesPatterns {
		for pattern, matchingHandlers := range index.nameToPatternMatchers[patternName] {
			patternsToMatchingHandlers[pattern] = matchingHandlers
		}
	}
	return patternsToMatchingHandlers
}

func getMapKeys(dict map[string]bool) []string {
	keys := make([]string, len(dict))
	i := 0
	for key := range dict {
		keys[i] = key
		i++
	}
	return keys
}
