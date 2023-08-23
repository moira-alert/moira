package filter

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	tagSpecRegex          = regexp.MustCompile(`^["']([^,!=]+)\s*(!?=~?)\s*([^"']*)["']`)
	tagSpecDelimiterRegex = regexp.MustCompile(`^\s*,\s*`)
	seriesByTagRegex      = regexp.MustCompile(`^seriesByTag\((.+)\)$`)
	wildcardExprRegex     = regexp.MustCompile(`\{(.*?)\}`)
)

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

	correctLengthOfMatchedWildcardIndexesSlice = 4
)

// TagSpec is a filter expression inside seriesByTag pattern
type TagSpec struct {
	Name     string
	Operator TagSpecOperator
	Value    string
}

// transformWildcardToRegexpInSeriesByTag is used to convert regular expression from graphite regexp format
// to standard regexp parsable by the go regexp engine.
func transformWildcardToRegexpInSeriesByTag(input string) (string, bool) {
	var isTransformed = false
	result := input

	if strings.Contains(result, "*") {
		result = strings.ReplaceAll(result, "*", ".*")
		isTransformed = true
	}

	for {
		matchedWildcardIndexes := wildcardExprRegex.FindStringSubmatchIndex(result)
		if len(matchedWildcardIndexes) != correctLengthOfMatchedWildcardIndexesSlice {
			break
		}

		wildcardExpression := result[matchedWildcardIndexes[0]:matchedWildcardIndexes[1]]
		regularExpression := strings.ReplaceAll(wildcardExpression, "{", "(")
		regularExpression = strings.ReplaceAll(regularExpression, "}", ")")
		slc := strings.Split(regularExpression, ",")
		for i := range slc {
			slc[i] = strings.TrimSpace(slc[i])
		}
		regularExpression = strings.Join(slc, "|")
		result = result[:matchedWildcardIndexes[0]] + regularExpression + result[matchedWildcardIndexes[1]:]
		isTransformed = true
	}

	if isTransformed {
		result = "^" + result + "$"
	}

	return result, isTransformed
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

		if operator != MatchOperator && operator != NotMatchOperator {
			var isTransformed bool

			// if got spec like '{a,b}' we must transform it to regex and change operator from 'equal' to 'match'
			if spec, isTransformed = transformWildcardToRegexpInSeriesByTag(spec); isTransformed {
				if operator == EqualOperator {
					operator = MatchOperator
				} else {
					operator = NotMatchOperator
				}
			}
		}

		tagSpecs = append(tagSpecs, TagSpec{name, operator, spec})
		input = input[matchedTagSpecIndexes[1]:]
	}

	return tagSpecs, nil
}

// MatchingHandler is a function for pattern matching
type MatchingHandler func(string, map[string]string) bool

// CreateMatchingHandlerForPattern creates function for matching by tag list
func CreateMatchingHandlerForPattern(tagSpecs []TagSpec) (string, MatchingHandler, error) {
	matchingHandlers := make([]MatchingHandler, 0)
	var nameTagValue string

	for _, tagSpec := range tagSpecs {
		if tagSpec.Name == "name" && tagSpec.Operator == EqualOperator {
			nameTagValue = tagSpec.Value
		} else {
			tagHandler, err := createMatchingHandlerForOneTag(tagSpec)
			if err != nil {
				return nameTagValue, nil, err
			}
			matchingHandlers = append(matchingHandlers, tagHandler)
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

	return nameTagValue, matchingHandler, nil
}

func createMatchingHandlerForOneTag(spec TagSpec) (MatchingHandler, error) {
	var matchingHandlerCondition func(string) bool
	switch spec.Operator {
	case EqualOperator:
		matchingHandlerCondition = func(value string) bool {
			return value == spec.Value
		}
	case NotEqualOperator:
		matchingHandlerCondition = func(value string) bool {
			return value != spec.Value
		}
	case MatchOperator:
		value := spec.Value
		// work around
		if value == "*" {
			value = ".*"
		}
		if !strings.HasPrefix(value, "^") {
			value = ".*" + value
		}
		if !strings.HasSuffix(value, "$") {
			value += ".*"
		}
		matchRegex, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
		matchingHandlerCondition = func(value string) bool {
			return matchRegex.MatchString(value)
		}
	case NotMatchOperator:
		value := spec.Value
		// work around
		if value == "*" {
			value = ".*"
		}
		if !strings.HasPrefix(value, "^") {
			value = ".*" + value
		}
		if !strings.HasSuffix(value, "$") {
			value += ".*"
		}
		matchRegex, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
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
		return matchEmpty
	}, nil
}
