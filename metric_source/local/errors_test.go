package local

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestErrDifferentPatternsTimeRanges_Error(t *testing.T) {
	Convey("ErrDifferentPatternsTimeRanges.Error", t, func() {
		err := ErrDifferentPatternsTimeRanges{
			patterns: []string{
				"test.pattern.1: from: -60, until: 0",
				"test.pattern.2: from: 0, until: 0",
			},
		}
		actual := err.Error()
		So(actual, ShouldResemble, "Some of patterns have different time ranges in the same target:\ntest.pattern.1: from: -60, until: 0\ntest.pattern.2: from: 0, until: 0")
	})
}

func Test_newErrDifferentPatternsTimeRangesBuilder(t *testing.T) {
	Convey("newErrDifferentPatternsTimeRangesBuilder", t, func() {
		builder := newErrDifferentPatternsTimeRangesBuilder()
		So(builder, ShouldResemble, errDifferentPatternsTimeRangesBuilder{
			result:      &ErrDifferentPatternsTimeRanges{},
			returnError: false,
		})
	})
}

func Test_errDifferentPatternsTimeRangesBuilder_addPattern(t *testing.T) {
	Convey("errDifferentPatternsTimeRangesBuilder.addPattern", t, func() {
		builder := newErrDifferentPatternsTimeRangesBuilder()
		builder.addPattern("test.pattern.1", -60, 0)
		So(builder, ShouldResemble, errDifferentPatternsTimeRangesBuilder{
			result: &ErrDifferentPatternsTimeRanges{
				patterns: []string{
					"test.pattern.1: from: -60, until: 0",
				},
			},
			returnError: true,
		})
	})
}

func Test_errDifferentPatternsTimeRangesBuilder_addCommon(t *testing.T) {
	Convey("errDifferentPatternsTimeRangesBuilder.addCommon", t, func() {
		builder := newErrDifferentPatternsTimeRangesBuilder()
		builder.addCommon("test.pattern.1", -60, 0)
		So(builder, ShouldResemble, errDifferentPatternsTimeRangesBuilder{
			result: &ErrDifferentPatternsTimeRanges{
				patterns: []string{
					"test.pattern.1: from: -60, until: 0",
				},
			},
			returnError: false,
		})
	})
}

func Test_errDifferentPatternsTimeRangesBuilder_build(t *testing.T) {
	Convey("errDifferentPatternsTimeRangesBuilder.build", t, func() {
		builder := newErrDifferentPatternsTimeRangesBuilder()
		Convey("error is not nil", func() {
			builder.addPattern("test.pattern.1", -60, 0)
			err := builder.build()
			So(err, ShouldResemble, ErrDifferentPatternsTimeRanges{
				patterns: []string{
					"test.pattern.1: from: -60, until: 0",
				},
			})
		})
		Convey("error is nil", func() {
			err := builder.build()
			So(err, ShouldBeNil)
		})
	})
}
