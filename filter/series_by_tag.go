package filter

import (
	"fmt"
	"regexp"
)

var tagSpecRegexString = "\"(?P<name>[^,!=]+)\\s*(?P<operator>!?=~?)\\s*(?P<spec>[^,]*)\""
var tagSpecsDelimiterRegexString = "\\s*,\\s*"
var tagSpecsRegexString = tagSpecRegexString + "(" + tagSpecsDelimiterRegexString + tagSpecRegexString + ")*"
var seriesByTagRegexString = "^seriesByTag\\(" + tagSpecsRegexString + "\\)$"
var seriesByTagRegex = regexp.MustCompile(seriesByTagRegexString)

var ErrNotSeriesByTag = fmt.Errorf("not seriesByTag pattern")

type TagSpecOperator string

const (
	Equal    TagSpecOperator = "="
	NotEqual TagSpecOperator = "!="
	Match    TagSpecOperator = "=~"
	NotMatch TagSpecOperator = "!=~"
)

type TagSpec struct {
	Name     string
	Operator TagSpecOperator
	Pattern  string
}

//ParseSeriesByTag parses seriesByTag pattern and returns tags queries
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
