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
			{"seriesByTag('a=b')", []TagSpec{{"a", EqualOperator, "b"}}},

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

	Convey("Given invalid seriesByTag patterns, should return error", t, func(c C) {
		invalidSeriesByTagCases := []string{
			"seriesByTag(\"a=b')",
			"seriesByTag('a=b\")",
		}

		for _, invalidCase := range invalidSeriesByTagCases {
			_, err := ParseSeriesByTag(invalidCase)
			c.So(err, ShouldNotBeNil)
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

			"name=cpu1;dc=ru1": {{"name", MatchOperator, "cpu1"}, {"dc", EqualOperator, "ru1"}},
			"name=cpu1;dc=ru2": {{"name", MatchOperator, "cpu1"}, {"dc", EqualOperator, "ru2"}},
			"name=cpu2;dc=ru1": {{"name", MatchOperator, "cpu2"}, {"dc", EqualOperator, "ru1"}},
			"name=cpu2;dc=ru2": {{"name", MatchOperator, "cpu2"}, {"dc", EqualOperator, "ru2"}},
			"name=disk;dc=ru1": {{"name", MatchOperator, "disk"}, {"dc", EqualOperator, "ru1"}},
			"name=disk;dc=ru2": {{"name", MatchOperator, "disk"}, {"dc", EqualOperator, "ru2"}},
			"name=cpu1;dc=us":  {{"name", MatchOperator, "cpu1"}, {"dc", EqualOperator, "us"}},
			"name=cpu2;dc=us":  {{"name", MatchOperator, "cpu2"}, {"dc", EqualOperator, "us"}},

			"name~=cpu;dc=":   {{"name", MatchOperator, "cpu"}, {"dc", EqualOperator, ""}},
			"name~=cpu;dc!=":  {{"name", MatchOperator, "cpu"}, {"dc", NotEqualOperator, ""}},
			"name~=cpu;dc~=":  {{"name", MatchOperator, "cpu"}, {"dc", MatchOperator, ""}},
			"name~=cpu;dc!~=": {{"name", MatchOperator, "cpu"}, {"dc", NotMatchOperator, ""}},
		}
		testCases := []struct {
			Name            string
			Labels          map[string]string
			MatchedPatterns []string
		}{
			{"cpu1", map[string]string{}, []string{"name=cpu1", "name~=cpu", "name~=cpu;dc=", "name~=cpu;dc~="}},
			{"cpu2", map[string]string{}, []string{"name!=cpu1", "name~=cpu", "name~=cpu;dc=", "name~=cpu;dc~="}},
			{"disk", map[string]string{}, []string{"name!=cpu1", "name!~=cpu"}},
			{"cpu1", map[string]string{"dc": "ru1"}, []string{"dc=ru1", "dc~=ru", "name=cpu1", "name=cpu1;dc=ru1", "name~=cpu", "name~=cpu;dc!=", "name~=cpu;dc~="}},
			{"cpu1", map[string]string{"dc": "ru2"}, []string{"dc!=ru1", "dc~=ru", "name=cpu1", "name=cpu1;dc=ru2", "name~=cpu", "name~=cpu;dc!=", "name~=cpu;dc~="}},
			{"cpu1", map[string]string{"dc": "us"}, []string{"dc!=ru1", "dc!~=ru", "name=cpu1", "name=cpu1;dc=us", "name~=cpu", "name~=cpu;dc!=", "name~=cpu;dc~="}},
			{"cpu1", map[string]string{"machine": "machine"}, []string{"name=cpu1", "name~=cpu", "name~=cpu;dc=", "name~=cpu;dc~="}},
		}

		index := NewSeriesByTagPatternIndex(tagSpecsByPattern)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})
}
