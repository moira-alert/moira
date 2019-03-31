package filter

import (
	"fmt"
	"regexp"

	"github.com/moira-alert/moira"
)

var tagSpecRegex = regexp.MustCompile(`^["']{1}([^,!=]+)\s*(!?=~?)\s*([^,]*)["']{1}`)
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

		name := input[matchedTagSpecIndexes[2]:matchedTagSpecIndexes[3]]
		operator := TagSpecOperator(input[matchedTagSpecIndexes[4]:matchedTagSpecIndexes[5]])
		spec := input[matchedTagSpecIndexes[6]:matchedTagSpecIndexes[7]]

		tagSpecs = append(tagSpecs, TagSpec{name, operator, spec})

		input = input[matchedTagSpecIndexes[1]:]
	}
	return tagSpecs, nil
}

// SeriesByTagPatternIndex helps to index the seriesByTag patterns and allows to match them by metric
type SeriesByTagPatternIndex struct {
	filtersByTag map[string][]func(string) ([]string, bool)
}

// NewSeriesByTagPatternIndex creates new SeriesByTagPatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag
func NewSeriesByTagPatternIndex(tagSpecsByPattern map[string][]TagSpec) *SeriesByTagPatternIndex {
	tagSpecsByTag := make(map[string]map[TagSpec][]string)

	for pattern, tagSpecs := range tagSpecsByPattern {
		for _, tagSpec := range tagSpecs {
			var patternsByTagSpec map[TagSpec][]string
			if value, ok := tagSpecsByTag[tagSpec.Name]; ok {
				patternsByTagSpec = value
			} else {
				patternsByTagSpec = make(map[TagSpec][]string)
			}

			patterns := patternsByTagSpec[tagSpec]
			patternsByTagSpec[tagSpec] = append(patterns, pattern)
			tagSpecsByTag[tagSpec.Name] = patternsByTagSpec
		}
	}

	filters := make(map[string][]func(string) ([]string, bool))

	for tag, tagSpecsGroup := range tagSpecsByTag {
		tagFilters := make([]func(string) ([]string, bool), 0)
		for tagSpec, patterns := range tagSpecsGroup {
			tagFilters = append(tagFilters, createFilter(tagSpec, patterns))
		}
		filters[tag] = tagFilters
	}

	return &SeriesByTagPatternIndex{filtersByTag: filters}
}

// MatchPatterns allows to match patterns by metric name and its labels
func (index *SeriesByTagPatternIndex) MatchPatterns(name string, labels map[string]string) []string {
	matchedPatterns := make([][]string, 0)

	if filters, found := index.filtersByTag["name"]; found {
		for _, filter := range filters {
			if patterns, matched := filter(name); matched {
				matchedPatterns = append(matchedPatterns, patterns)
			}
		}
	}

	for name, value := range labels {
		if filters, found := index.filtersByTag[name]; found {
			for _, filter := range filters {
				if patterns, matched := filter(value); matched {
					matchedPatterns = append(matchedPatterns, patterns)
				}
			}
		}
	}

	return moira.GetStringListsUnion(matchedPatterns...)
}

func createFilter(spec TagSpec, patterns []string) func(string) ([]string, bool) {
	var filterCondition func(string) bool
	switch spec.Operator {
	case EqualOperator:
		filterCondition = func(value string) bool {
			return value == spec.Value
		}
	case NotEqualOperator:
		filterCondition = func(value string) bool {
			return value != spec.Value
		}
	case MatchOperator:
		matchRegex := regexp.MustCompile("^" + spec.Value)
		filterCondition = func(value string) bool {
			return matchRegex.MatchString(value)
		}
	case NotMatchOperator:
		matchRegex := regexp.MustCompile("^" + spec.Value)
		filterCondition = func(value string) bool {
			return !matchRegex.MatchString(value)
		}
	default:
		filterCondition = func(value string) bool {
			return false
		}
	}

	return func(value string) ([]string, bool) {
		if filterCondition(value) {
			return patterns, true
		}

		return nil, false
	}
}
