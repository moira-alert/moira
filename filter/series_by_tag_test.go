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
			{"seriesByTag(\"a=b\")", []TagSpec{{"a", Equal, "b"}}},
			{"seriesByTag(\"a!=b\")", []TagSpec{{"a", NotEqual, "b"}}},
			{"seriesByTag(\"a=~b\")", []TagSpec{{"a", Match, "b"}}},
			{"seriesByTag(\"a!=~b\")", []TagSpec{{"a", NotMatch, "b"}}},
			{"seriesByTag(\"a=\")", []TagSpec{{"a", Equal, ""}}},
			{"seriesByTag(\"a=b\",\"a=c\")", []TagSpec{{"a", Equal, "c"}}},
		}

		for _, validCase := range validSeriesByTagCases {
			specs, err := ParseSeriesByTag(validCase.pattern)
			So(err, ShouldBeNil)
			So(specs, ShouldResemble, validCase.tagSpecs)
		}
	})
}
