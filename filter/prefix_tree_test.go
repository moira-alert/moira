package filter

import (
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPrefixTree(t *testing.T) {
	logger, _ := logging.GetLogger("PrefixTree")

	Convey("Working with empty tree", t, func() {
		prefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}

		Convey("Match should return empty string array", func() {
			matchedPatterns := prefixTree.Match("any_string")
			So(matchedPatterns, ShouldResemble, []string{})
		})

		Convey("MatchWithValue should return empty map", func() {
			matchedPatterns := prefixTree.MatchWithValue("any_string")
			So(matchedPatterns, ShouldResemble, map[string]MatchingHandler{})
		})
	})

	Convey("Working with tree without payload", t, func() {
		prefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}

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

		for _, pattern := range patterns {
			prefixTree.Add(pattern)
		}

		Convey("Metrics from tree should be matched", func() {
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
			}
			for _, testCase := range testCases {
				matchedPatterns := prefixTree.Match(testCase.Metric)
				So(matchedPatterns, ShouldResemble, testCase.MatchedPatterns)
			}
		})

		Convey("Metrics not from tree should return empty string array", func() {
			testCases := []string{
				"Two.dots..together",
				"Simple.notmatching.pattern",
				"Star.nothing",
				"Bracket.one.nothing",
				"Bracket.nothing.pattern",
				"Complex.prefixonesuffix",
			}
			for _, testCase := range testCases {
				matchedPatterns := prefixTree.Match(testCase)
				So(matchedPatterns, ShouldResemble, []string{})
			}
		})

		Convey("MatchWithValue should return map with empty values", func() {
			Convey("For metrics from tree", func() {
				metric := "Complex.matching.pattern"
				matchedValue := map[string]MatchingHandler{
					"Complex.matching.pattern": nil,
					"Complex.*.*":              nil,
				}
				matchedPatterns := prefixTree.MatchWithValue(metric)
				So(matchedPatterns, ShouldResemble, matchedValue)
			})

			Convey("For metrics not from tree", func() {
				metric := "Simple.notmatching.pattern"
				matchedValue := map[string]MatchingHandler{}
				matchedPatterns := prefixTree.MatchWithValue(metric)
				So(matchedPatterns, ShouldResemble, matchedValue)
			})
		})
	})

	Convey("Working with tree with payload", t, func() {
		prefixTree := &PrefixTree{Logger: logger, Root: &PatternNode{}}

		trueHandler := func(string, map[string]string) bool { return true }
		falseHandler := func(string, map[string]string) bool { return false }

		patterns := []struct {
			Pattern      string
			PayloadKey   string
			PayloadValue MatchingHandler
		}{
			{"Simple.matching.pattern", "Simple.matching.pattern;k1", trueHandler},
			{"Simple.matching.pattern.*", "Simple.matching.pattern.*;k1", trueHandler},
			{"Simple.matching.pattern.*", "Simple.matching.pattern.*;k2", falseHandler},
		}

		for _, pattern := range patterns {
			prefixTree.AddWithPayload(pattern.Pattern, pattern.PayloadKey, pattern.PayloadValue)
		}

		Convey("Metrics from tree should be matched", func() {
			testCases := []struct {
				Metric          string
				MatchedPatterns map[string]MatchingHandler
			}{
				{"Simple.matching.pattern", map[string]MatchingHandler{
					"Simple.matching.pattern;k1": trueHandler,
				}},
				{"Simple.matching.pattern.*", map[string]MatchingHandler{
					"Simple.matching.pattern.*;k1": trueHandler,
					"Simple.matching.pattern.*;k2": falseHandler,
				}},
			}
			for _, testCase := range testCases {
				matchedPatterns := prefixTree.MatchWithValue(testCase.Metric)
				So(len(matchedPatterns), ShouldEqual, len(testCase.MatchedPatterns))
				for pKey, pValue := range testCase.MatchedPatterns {
					So(matchedPatterns, ShouldContainKey, pKey)
					if pValue == nil {
						So(matchedPatterns[pKey], ShouldBeNil)
					} else {
						So(matchedPatterns[pKey]("", nil), ShouldEqual, testCase.MatchedPatterns[pKey]("", nil))
					}
				}
			}
		})

		Convey("Metrics not from tree should return empty map", func() {
			testCases := []string{
				"Two.dots..together",
				"Simple.notmatching.pattern",
			}
			for _, testCase := range testCases {
				matchedPatterns := prefixTree.MatchWithValue(testCase)
				So(matchedPatterns, ShouldResemble, map[string]MatchingHandler{})
			}
		})

		Convey("Match should return matched patterns without Payload", func() {
			Convey("For metrics from tree", func() {
				metric := "Simple.matching.pattern"
				matchedValue := []string{
					"Simple.matching.pattern",
				}
				matchedPatterns := prefixTree.Match(metric)
				So(matchedPatterns, ShouldResemble, matchedValue)
			})

			Convey("For metrics not from tree", func() {
				metric := "Simple.notmatching.pattern"
				matchedValue := []string{}
				matchedPatterns := prefixTree.Match(metric)
				So(matchedPatterns, ShouldResemble, matchedValue)
			})
		})
	})
}
