package filter

import (
	"testing"

	"github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPatternIndex(t *testing.T) {
	logger, _ := logging.GetLogger("PatternIndex")

	Convey("Given patterns, should build index and match patterns", t, func() {
		patterns := []string{
			"Simple.matching.pattern",
			"Simple.matching.pattern.*",
			"Star.single.*",
			"Star.*.double.any*",
			"Bracket.{one,two,three}.pattern",
			"Bracket.pr{one,two,three}suf",
			"Complex.matching.pattern",
			"Complex.*.*",
			"Complex.*.",
			"Complex.*{one,two,three}suf*.pattern",
			"Question.?at_begin",
			"Question.at_the_end?",
		}

		index := NewPatternIndex(logger, patterns)
		testCases := []struct {
			Metric          string
			MatchedPatterns []string
		}{
			{"Simple.matching.pattern", []string{"Simple.matching.pattern"}},
			{"Star.single.anything", []string{"Star.single.*"}},
			{"Star.anything.double.anything", []string{"Star.*.double.any*"}},
			{"Bracket.one.pattern", []string{"Bracket.{one,two,three}.pattern"}},
			{"Bracket.two.pattern", []string{"Bracket.{one,two,three}.pattern"}},
			{"Bracket.three.pattern", []string{"Bracket.{one,two,three}.pattern"}},
			{"Bracket.pronesuf", []string{"Bracket.pr{one,two,three}suf"}},
			{"Bracket.prtwosuf", []string{"Bracket.pr{one,two,three}suf"}},
			{"Bracket.prthreesuf", []string{"Bracket.pr{one,two,three}suf"}},
			{"Complex.matching.pattern", []string{"Complex.matching.pattern", "Complex.*.*"}},
			{"Complex.anything.pattern", []string{"Complex.*.*"}},
			{"Complex.prefixonesuffix.pattern", []string{"Complex.*.*", "Complex.*{one,two,three}suf*.pattern"}},
			{"Complex.prefixtwofix.pattern", []string{"Complex.*.*"}},
			{"Complex.anything.pattern", []string{"Complex.*.*"}},
			{"Question.1at_begin", []string{"Question.?at_begin"}},
			{"Question.at_the_end2", []string{"Question.at_the_end?"}},
			{"Two.dots..together", []string{}},
			{"Simple.notmatching.pattern", []string{}},
			{"Star.nothing", []string{}},
			{"Bracket.one.nothing", []string{}},
			{"Bracket.nothing.pattern", []string{}},
			{"Complex.prefixonesuffix", []string{}},
		}

		for _, testCase := range testCases {
			matchedPatterns := index.MatchPatterns(testCase.Metric)
			So(matchedPatterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})
}
