package main

import (
	"fmt"

	"github.com/moira-alert/moira/filter"
)

type Compatibility struct {
	// Controls how regexps are treated.
	// The default of "strict_start_match" treats /<regex>/ as /^<regex>.*$/
	// "loose_start_match" treats /<regex>/ as /^.*<regex>.*$/
	RegexTreatment string `yaml:"regex_treatment"`
}

const (
	strictStartMatchString string = "strict_start_match"
	looseStartMatchString  string = "loose_start_match"
)

func (compatibility *Compatibility) ToFilterCompatibility() (filter.Compatibility, error) {
	regex := compatibility.RegexTreatment

	if regex == strictStartMatchString || regex == "" {
		return filter.Compatibility{RegexTreatment: filter.StrictStartMatch}, nil
	}

	if regex == looseStartMatchString {
		return filter.Compatibility{RegexTreatment: filter.LooseStartMatch}, nil
	}

	return filter.Compatibility{}, fmt.Errorf(
		"cannot unmarshal `%s` into RegexTreatment, expected either `%s` or `%s`",
		regex, strictStartMatchString, looseStartMatchString,
	)
}
