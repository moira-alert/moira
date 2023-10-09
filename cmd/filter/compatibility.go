package main

import (
	"github.com/moira-alert/moira/filter"
)

// Compatibility struct contains feature-flags that give user control over
// features supported by different versions of graphit compatible with moira
type compatibility struct {
	// Controls how regices in tag matching are treated
	// If false (default value), regex will match start of the string strictly. 'tag~=foo' is equivalent to 'tag~=^foo.*'
	// If true, regex will match start of the string loosely. 'tag~=foo' is equivalent to 'tag~=.*foo.*'
	AllowRegexLooseStartMatch bool `yaml:"allow_regex_loose_start_match"`
	// Controls how absent tags are treated
	// If true (default value), empty tags in regices will be matched
	// If false, empty tags will be discarded
	AllowRegexMatchEmpty bool `yaml:"allow_regex_match_empty"`
}

func (compatibility *compatibility) toFilterCompatibility() filter.Compatibility {
	return filter.Compatibility{
		AllowRegexLooseStartMatch: compatibility.AllowRegexLooseStartMatch,
		AllowRegexMatchEmpty:      compatibility.AllowRegexMatchEmpty,
	}
}
