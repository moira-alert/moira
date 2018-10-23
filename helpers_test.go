package moira

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSubset(t *testing.T) {
	Convey("Test subsets", t, func() {
		So(Subset([]string{"1", "2", "3"}, []string{"3", "2", "1"}), ShouldBeTrue)
		So(Subset([]string{"1", "2", "3"}, []string{"1", "1", "1", "2", "2", "2", "3", "3", "3"}), ShouldBeTrue)
		So(Subset([]string{"1", "2", "3"}, []string{"123", "2", "3"}), ShouldBeFalse)
		So(Subset([]string{"1", "2", "3"}, []string{"1", "2", "4"}), ShouldBeFalse)
	})
}

func TestLeftJoin(t *testing.T) {
	Convey("Test left Join", t, func() {
		left := []string{"1", "2", "3"}
		right := []string{"1", "2", "3"}
		joined := LeftJoinStrings(left, right)
		So(joined, ShouldResemble, []string{})

		left = []string{"1", "2", "3", "4", "5"}
		joined = LeftJoinStrings(left, right)
		So(joined, ShouldResemble, []string{"4", "5"})

		right = []string{"6", "7", "8"}
		joined = LeftJoinStrings(left, right)
		So(joined, ShouldResemble, left)
	})
}
