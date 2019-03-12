package filter

import (
	"fmt"
	"regexp"

	"github.com/moira-alert/moira"
)

var tagSpecRegexString = "\"(?P<name>[^,!=]+)\\s*(?P<operator>!?=~?)\\s*(?P<spec>[^,]*)\""
var tagSpecsDelimiterRegexString = "\\s*,\\s*"
var tagSpecsRegexString = tagSpecRegexString + "(" + tagSpecsDelimiterRegexString + tagSpecRegexString + ")*"
var seriesByTagRegexString = "^seriesByTag\\(" + tagSpecsRegexString + "\\)$"
var seriesByTagRegex = regexp.MustCompile(seriesByTagRegexString)

//ErrNotSeriesByTag is returned if the pattern is not seriesByTag
var ErrNotSeriesByTag = fmt.Errorf("not seriesByTag pattern")

//TagSpecOperator is an operator inside a TagSpec
type TagSpecOperator string

const (
	Equal    TagSpecOperator = "="
	NotEqual TagSpecOperator = "!="
	Match    TagSpecOperator = "=~"
	NotMatch TagSpecOperator = "!=~"
)

//TagSpec is a filter expression inside seriesByTag pattern
type TagSpec struct {
	Name     string
	Operator TagSpecOperator
	Value    string
}

//ParseSeriesByTag parses seriesByTag pattern and returns tags specs
func ParseSeriesByTag(pattern string) ([]TagSpec, error) {
	matches := seriesByTagRegex.FindStringSubmatch(pattern)
	if len(matches) == 0 {
		return nil, ErrNotSeriesByTag
	}

	tagSpecsByName := make(map[string]TagSpec)
	subExprNames := seriesByTagRegex.SubexpNames()

	index := 0

	for index < len(subExprNames) {
		subExprName := subExprNames[index]
		if subExprName != "name" {
			index++
			continue
		}

		name := matches[index]
		if len(name) == 0 {
			break
		}

		operator := TagSpecOperator(matches[index+1])
		spec := matches[index+2]
		index += 3
		tagSpecsByName[name] = TagSpec{name, operator, spec}
	}

	tagSpecs := make([]TagSpec, 0, len(tagSpecsByName))
	for _, value := range tagSpecsByName {
		tagSpecs = append(tagSpecs, value)
	}

	return tagSpecs, nil
}

//SeriesByTagIndex helps to index the seriesByTag patterns and allows to match them by metric
type SeriesByTagIndex struct {
	filters map[string][]func(string) ([]string, bool)
}

// NewPatternIndex creates new PatternIndex using seriesByTag patterns and parsed specs comes from ParseSeriesByTag
func NewSeriesByTagIndex(tagSpecsByPattern map[string][]TagSpec) *SeriesByTagIndex {
	tagSpecsByTag := make(map[string]map[TagSpec][]string)

	for pattern, tagSpecs := range tagSpecsByPattern {
		for _, tagSpec := range tagSpecs {
			var patternsByTagSpec map[TagSpec][]string
			if value, ok := tagSpecsByTag[tagSpec.Name]; ok {
				patternsByTagSpec = value
			} else {
				patternsByTagSpec = make(map[TagSpec][]string)
			}

			patterns, _ := patternsByTagSpec[tagSpec]
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

	return &SeriesByTagIndex{filters: filters}
}

//MatchPatterns allows to match patterns by metric name and its labels
func (index *SeriesByTagIndex) MatchPatterns(name string, labels map[string]string) []string {
	matchedPatterns := make([][]string, 0)

	if filters, found := index.filters["name"]; found {
		for _, filter := range filters {
			if patterns, matched := filter(name); matched {
				matchedPatterns = append(matchedPatterns, patterns)
			}
		}
	}

	for name, value := range labels {
		if filters, found := index.filters[name]; found {
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
	case Equal:
		filterCondition = func(value string) bool {
			return value == spec.Value
		}
	case NotEqual:
		filterCondition = func(value string) bool {
			return value != spec.Value
		}
	case Match:
		matchRegex := regexp.MustCompile("^" + spec.Value)
		filterCondition = func(value string) bool {
			return matchRegex.MatchString(value)
		}
	case NotMatch:
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
		} else {
			return nil, false
		}
	}
}
