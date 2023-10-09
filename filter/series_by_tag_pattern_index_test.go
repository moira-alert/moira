package filter

import (
	"sort"
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTransformTaggedWildCardToMatchOperator(t *testing.T) {
	Convey("Given valid seriesByTag patterns, should return parsed tag specs", t, func() {
		testCases := []struct {
			PatternWithWildcard string
			PatternWithRegexp   string
			IsTransformed       bool
		}{
			{
				`{405,406,407,411,413,414,415}`,
				`^(405|406|407|411|413|414|415)$`,
				true,
			},
			{
				`aaa.{405,406,407,411,413,414,415}.bbb`,
				`^aaa\.(405|406|407|411|413|414|415)\.bbb$`,
				true,
			},
			{
				`aaa.{405,406}.bbb.{301,302}`,
				`^aaa\.(405|406)\.bbb\.(301|302)$`,
				true,
			},
			{
				`aaa.bbb*`,
				`^aaa\.bbb.*$`,
				true,
			},
			{
				`aaa.bbb.*`,
				`^aaa\.bbb\..*$`,
				true,
			},
			{
				`a(b|c|d)e`,
				`a(b|c|d)e`,
				false,
			},
			{
				`a.e`,
				`a.e`,
				false,
			},
		}

		for _, testCase := range testCases {
			result, isTransformed := transformWildcardToRegexpInSeriesByTag(testCase.PatternWithWildcard)
			So(result, ShouldEqual, testCase.PatternWithRegexp)
			So(isTransformed, ShouldEqual, testCase.IsTransformed)
		}
	})
}

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
			{`seriesByTag("a=b","a=c")`, []TagSpec{{"a", EqualOperator, "b"}, {"a", EqualOperator, "c"}}},
			{`seriesByTag("a=b","b=c","c=d")`, []TagSpec{{"a", EqualOperator, "b"}, {"b", EqualOperator, "c"}, {"c", EqualOperator, "d"}}},
			{`seriesByTag("a={b,c,d}")`, []TagSpec{{"a", MatchOperator, "^(b|c|d)$"}}},
			{`seriesByTag("a=~aa.(b|c|d)$")`, []TagSpec{{"a", MatchOperator, "aa.(b|c|d)$"}}},
			{`seriesByTag("respCode=~^(4|5)\d{2}")`, []TagSpec{{"respCode", MatchOperator, "^(4|5)\\d{2}"}}},
			{`seriesByTag("a={b,c,d}", "e=f")`, []TagSpec{{"a", MatchOperator, "^(b|c|d)$"}, {"e", EqualOperator, "f"}}},
			{`seriesByTag("a!={b,c,d}", "e=f")`, []TagSpec{{"a", NotMatchOperator, "^(b|c|d)$"}, {"e", EqualOperator, "f"}}},
			{`seriesByTag('a!={b,c,d}', 'e=f')`, []TagSpec{{"a", NotMatchOperator, "^(b|c|d)$"}, {"e", EqualOperator, "f"}}},
			{`seriesByTag('a=b*', 'e=f')`, []TagSpec{{"a", MatchOperator, "^b.*$"}, {"e", EqualOperator, "f"}}},
			{`seriesByTag('a=b.*')`, []TagSpec{{"a", MatchOperator, "^b\\..*$"}}},
			{`seriesByTag('a=b.c')`, []TagSpec{{"a", EqualOperator, "b.c"}}},
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
	var logger, _ = logging.GetLogger("SeriesByTag")
	Convey("Given empty patterns with tagspecs, should build index and match patterns", t, func(c C) {
		compatibility := Compatibility{
			RegexTreatment: LooseStartMatch,
		}
		index := NewSeriesByTagPatternIndex(logger, map[string][]TagSpec{}, compatibility)
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

			"name=cpu1;dc=ru1": {{"name", EqualOperator, "cpu1"}, {"dc", EqualOperator, "ru1"}},
			"name=cpu1;dc=ru2": {{"name", EqualOperator, "cpu1"}, {"dc", EqualOperator, "ru2"}},
			"name=cpu2;dc=ru1": {{"name", EqualOperator, "cpu2"}, {"dc", EqualOperator, "ru1"}},
			"name=cpu2;dc=ru2": {{"name", EqualOperator, "cpu2"}, {"dc", EqualOperator, "ru2"}},
			"name=disk;dc=ru1": {{"name", EqualOperator, "disk"}, {"dc", EqualOperator, "ru1"}},
			"name=disk;dc=ru2": {{"name", EqualOperator, "disk"}, {"dc", EqualOperator, "ru2"}},
			"name=cpu1;dc=us":  {{"name", EqualOperator, "cpu1"}, {"dc", EqualOperator, "us"}},
			"name=cpu2;dc=us":  {{"name", EqualOperator, "cpu2"}, {"dc", EqualOperator, "us"}},

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

		compatibility := Compatibility{
			RegexTreatment:  LooseStartMatch,
			SingleStarMatch: SingleStarMatchAllExisting,
		}
		index := NewSeriesByTagPatternIndex(logger, tagSpecsByPattern, compatibility)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})

	Convey("Given related patterns with tagspecs, should build index and match patterns", t, func(c C) {
		tagSpecsByPattern := map[string][]TagSpec{
			"name=cpu.test1.test2": {{"name", EqualOperator, "cpu.test1.test2"}},
			"name=cpu.*.test2":     {{"name", EqualOperator, "cpu.*.test2"}},
			"name=cpu.test1.*":     {{"name", EqualOperator, "cpu.test1.*"}},
			"name=cpu.*.*":         {{"name", EqualOperator, "cpu.*.*"}},

			"name=cpu.*.test2;tag1=val1": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag1", EqualOperator, "val1"},
			},
			"name=cpu.*.test2;tag2=val2": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag2", EqualOperator, "val2"},
			},
			"name=cpu.*.test2;tag1=val1;tag2=val2": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
			"name!=cpu.test1.test2;tag1=val1;tag2=val2": {
				{"name", NotEqualOperator, "cpu.test1.test2"},
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
			"name=~cpu;tag1=val1": {
				{"name", MatchOperator, "cpu"},
				{"tag1", EqualOperator, "val1"},
			},
			"name=~test1": {
				{"name", MatchOperator, "test1"},
			},
			"tag1=~al1": {
				{"tag1", MatchOperator, "al1"},
			},
			"tag2=~*": {
				{"tag2", MatchOperator, "*"},
			},
			"tag1=val1;tag2=val2": {
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
		}

		testCases := []struct {
			Name            string
			Labels          map[string]string
			MatchedPatterns []string
		}{
			{"cpu.test1.test2",
				map[string]string{},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~test1",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag": "val"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~test1",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"name=~test1",
					"tag1=~al1",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val2"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~test1",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1", "tag2": "val1"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"name=~test1",
					"tag1=~al1",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag2": "val2"},
				[]string{"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag2=val2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~test1",
					"tag2=~*",
				}},
			{"cpu.test3.test2",
				map[string]string{"tag2": "val2"},
				[]string{"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag2=val2",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1", "tag2": "val2"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.*.test2;tag1=val1;tag2=val2",
					"name=cpu.*.test2;tag2=val2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"name=~test1",
					"tag1=val1;tag2=val2",
					"tag1=~al1",
					"tag2=~*",
				}},
		}

		compatibility := Compatibility{
			RegexTreatment:  LooseStartMatch,
			SingleStarMatch: SingleStarMatchAllExisting,
		}
		index := NewSeriesByTagPatternIndex(logger, tagSpecsByPattern, compatibility)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})
}

func TestSeriesByTagPatternIndexCabonCompatibility(t *testing.T) {
	var logger, _ = logging.GetLogger("SeriesByTag")

	Convey("Given related patterns with tagspecs, should build index and match patterns", t, func(c C) {
		tagSpecsByPattern := map[string][]TagSpec{
			"name=cpu.test1.test2": {{"name", EqualOperator, "cpu.test1.test2"}},
			"name=cpu.*.test2":     {{"name", EqualOperator, "cpu.*.test2"}},
			"name=cpu.test1.*":     {{"name", EqualOperator, "cpu.test1.*"}},
			"name=cpu.*.*":         {{"name", EqualOperator, "cpu.*.*"}},

			"name=cpu.*.test2;tag1=val1": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag1", EqualOperator, "val1"},
			},
			"name=cpu.*.test2;tag2=val2": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag2", EqualOperator, "val2"},
			},
			"name=cpu.*.test2;tag1=val1;tag2=val2": {
				{"name", EqualOperator, "cpu.*.test2"},
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
			"name!=cpu.test1.test2;tag1=val1;tag2=val2": {
				{"name", NotEqualOperator, "cpu.test1.test2"},
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
			"name=~cpu;tag1=val1": {
				{"name", MatchOperator, "cpu"},
				{"tag1", EqualOperator, "val1"},
			},
			"name=~test1": {
				{"name", MatchOperator, "test1"},
			},
			"tag1=~al1": {
				{"tag1", MatchOperator, "al1"},
			},
			"tag2=~*": {
				{"tag2", MatchOperator, "*"},
			},
			"tag1=val1;tag2=val2": {
				{"tag1", EqualOperator, "val1"},
				{"tag2", EqualOperator, "val2"},
			},
		}

		testCases := []struct {
			Name            string
			Labels          map[string]string
			MatchedPatterns []string
		}{
			{"cpu.test1.test2",
				map[string]string{},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag": "val"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val2"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1", "tag2": "val1"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag2": "val2"},
				[]string{"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag2=val2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"tag2=~*",
				}},
			{"cpu.test3.test2",
				map[string]string{"tag2": "val2"},
				[]string{"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag2=val2",
					"tag2=~*",
				}},
			{"cpu.test1.test2",
				map[string]string{"tag1": "val1", "tag2": "val2"},
				[]string{
					"name=cpu.*.*",
					"name=cpu.*.test2",
					"name=cpu.*.test2;tag1=val1",
					"name=cpu.*.test2;tag1=val1;tag2=val2",
					"name=cpu.*.test2;tag2=val2",
					"name=cpu.test1.*",
					"name=cpu.test1.test2",
					"name=~cpu;tag1=val1",
					"tag1=val1;tag2=val2",
					"tag2=~*",
				}},
		}

		compatibility := Compatibility{
			RegexTreatment: StrictStartMatch,
		}
		index := NewSeriesByTagPatternIndex(logger, tagSpecsByPattern, compatibility)
		for _, testCase := range testCases {
			patterns := index.MatchPatterns(testCase.Name, testCase.Labels)
			sort.Strings(patterns)
			c.So(patterns, ShouldResemble, testCase.MatchedPatterns)
		}
	})
}
