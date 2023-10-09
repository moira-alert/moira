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
	// Controls how the match expression with single star works
	// The deafult value of "disabled" disbles this feature
	// "match_all_existing" makes 'some_tag~=*' match every metrics that has tag 'some_tag'
	SingleStarMatch string `yaml:"single_star_match"`
}

var regexTreatmentMap = map[string]filter.RegexTreatment{
	"":                   filter.StrictStartMatch,
	"strict_start_match": filter.StrictStartMatch,
	"loose_start_match":  filter.LooseStartMatch,
}

var singleStarMatchMap = map[string]filter.SingleStarMatch{
	"":                   filter.SingleStarMatchDisabled,
	"disabled":           filter.SingleStarMatchDisabled,
	"match_all_existing": filter.SingleStarMatchAllExisting,
}

func (compatibility *Compatibility) ToFilterCompatibility() (filter.Compatibility, error) {
	regexTreatment, ok := regexTreatmentMap[compatibility.RegexTreatment]
	if !ok {
		err := fmt.Errorf("cannot unmarshal `%s` into RegexTreatment", compatibility.RegexTreatment)
		return filter.Compatibility{}, err
	}

	singleStarMatch, ok := singleStarMatchMap[compatibility.SingleStarMatch]
	if !ok {
		err := fmt.Errorf("cannot unmarshal `%s` into SingleStarMatch", compatibility.SingleStarMatch)
		return filter.Compatibility{}, err
	}

	return filter.Compatibility{
		RegexTreatment:  regexTreatment,
		SingleStarMatch: singleStarMatch,
	}, nil
}
