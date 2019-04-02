package filter

import (
	"sort"
	"testing"
)
import . "github.com/smartystreets/goconvey/convey"

func TestParseSeriesByTag(t *testing.T) {
	type ValidSeriesByTagCase struct {
		pattern  string
		tagSpecs []TagSpec
	}

	Convey("Given valid seriesByTag patterns, should return parsed tag specs", t, func(c C) {
		validSeriesByTagCases := []ValidSeriesByTagCase{
			{"seriesByTag(\"a=b\")", []TagSpec{{"a", EqualOperator, "b"}}},
			{"seriesByTag(\"a!=b\")", []TagSpec{{"a", NotEqualOperator, "b"}}},
			{"seriesByTag(\"a=~b\")", []TagSpec{{"a", MatchOperator, "b"}}},
			{"seriesByTag(\"a!=~b\")", []TagSpec{{"a", NotMatchOperator, "b"}}},
			{"seriesByTag(\"a=\")", []TagSpec{{"a", EqualOperator, ""}}},
			{"seriesByTag(\"a=b\",\"a=c\")", []TagSpec{{"a", EqualOperator, "b"}, {"a", EqualOperator, "c"}}},
			{"seriesByTag(\"a=b\",\"b=c\",\"c=d\")", []TagSpec{{"a", EqualOperator, "b"}, {"b", EqualOperator, "c"}, {"c", EqualOperator, "d"}}},
		}

		for _, validCase := range validSeriesByTagCases {
			specs, err := ParseSeriesByTag(validCase.pattern)
			c.So(err, ShouldBeNil)
			c.So(specs, ShouldResemble, validCase.tagSpecs)
		}
	})
}

func TestSeriesByTagPatternIndex(t *testing.T) {
	Convey("Given empty patterns with tagspecs, should build index and match patterns", t, func(c C) {
		index := NewSeriesByTagPatternIndex(map[string][]TagSpec{})
		c.So(index.MatchPatterns("", nil), ShouldResemble, []string{})
	})

	Convey("Given simple patterns with tagspecs, should build index and match patterns", t, func(c C) {
		tagSpecsByPattern := map[string][]TagSpec{
			"name=cpu1":        {{"name", EqualOperator, "cpu1"}},
			"name!=cpu1":       {{"name", NotEqualOperator, "cpu1"}},
			"name~=cpu":        {{"name", MatchOperator, "cpu"}},
			"name!~=cpu":       {{"name", NotMatchOperator, "cpu"}},
			"dc=ru1":           {{"dc", EqualOperator, "ru1"}},
			"dc!=ru1":          {{"dc", NotEqualOperator, "ru1"}},
			"dc~=ru":           {{"dc", MatchOperator, "ru"}},
			"dc!~=ru":          {{"dc", NotMatchOperator, "ru"}},
			"invalid operator": {{"dc", TagSpecOperator("invalid operator"), "ru"}},
		}
		testCases := []struct {
			Name            string
			Labels          map[string]string
			MatchedPatterns []string
		}{
			{"cpu1", map[string]string{}, []string{"name=cpu1", "name~=cpu"}},
			{"cpu2", map[string]string{}, []string{"name!=cpu1", "name~=cpu"}},
			{"disk", map[string]string{}, []string{"name!=cpu1", "name!~=cpu"}},
			{"cpu1", map[string]string{"dc": "ru1"}, []string{"dc=ru1", "dc~=ru", "name=cpu1", "name~=cpu"}},
			{"cpu1", map[string]string{"dc": "ru2"}, []string{"dc!=ru1", "dc~=ru", "name=cpu1", "name~=cpu"}},
			{"cpu1", map[string]string{"dc": "us"}, []string{"dc!=ru1", "dc!~=ru", "name=cpu1", "name~=cpu"}},
			{"cpu1", map[string]string{"machine": "machine"}, []string{"name=cpu1", "name~=cpu"}},
		}

		index := NewSeriesByTagPatternIndex(tagSpecsByPattern)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})

	Convey("Given complex patterns with tagspecs, should build index and match patterns", t, func(c C) {
		tagSpecsByPattern := map[string][]TagSpec{
			"name=cpu1;dc=dc1": {{"name", EqualOperator, "cpu1"}, {"dc", EqualOperator, "dc1"}},
			"name=cpu2;dc=dc2": {{"name", EqualOperator, "cpu2"}, {"dc", EqualOperator, "dc2"}},
		}

		testCases := []struct {
			Name            string
			Labels          map[string]string
			MatchedPatterns []string
		}{
			{"cpu1", map[string]string{"dc": "dc1"}, []string{"name=cpu1;dc=dc1"}},
			{"cpu2", map[string]string{"dc": "dc2"}, []string{"name=cpu2;dc=dc2"}},
			{"cpu1", map[string]string{"dc": "dc2"}, []string{}},
			{"cpu2", map[string]string{"dc": "dc1"}, []string{}},
		}

		index := NewSeriesByTagPatternIndex(tagSpecsByPattern)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})
}
