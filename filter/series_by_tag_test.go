package filter

import "testing"
import . "github.com/smartystreets/goconvey/convey"

func TestParseSeriesByTag(t *testing.T) {
	type ValidSeriesByTagCase struct {
		pattern  string
		tagSpecs []TagSpec
	}

	Convey("Given valid seriesByTag patterns, should return parsed tag specs", t, func() {
		validSeriesByTagCases := []ValidSeriesByTagCase{
			{"seriesByTag(\"a=b\")", []TagSpec{{"a", EqualOperator, "b"}}},
			{"seriesByTag(\"a!=b\")", []TagSpec{{"a", NotEqualOperator, "b"}}},
			{"seriesByTag(\"a=~b\")", []TagSpec{{"a", MatchOperator, "b"}}},
			{"seriesByTag(\"a!=~b\")", []TagSpec{{"a", NotMatchOperator, "b"}}},
			{"seriesByTag(\"a=\")", []TagSpec{{"a", EqualOperator, ""}}},
			{"seriesByTag(\"a=b\",\"a=c\")", []TagSpec{{"a", EqualOperator, "c"}}},
		}

		for _, validCase := range validSeriesByTagCases {
			specs, err := ParseSeriesByTag(validCase.pattern)
			So(err, ShouldBeNil)
			So(specs, ShouldResemble, validCase.tagSpecs)
		}
	})
}
