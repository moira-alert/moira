package msgformat

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDefaultTagsLimiter(t *testing.T) {
	Convey("Test default tags limiter", t, func() {
		tags := []string{"tag1", "tag2"}

		Convey("with maxSize < 0", func() {
			tagsStr := DefaultTagsLimiter(tags, -1)

			So(tagsStr, ShouldResemble, "")
		})

		Convey("with maxSize > total characters in tags string", func() {
			tagsStr := DefaultTagsLimiter(tags, 30)

			So(tagsStr, ShouldResemble, " [tag1][tag2]")
		})

		Convey("with maxSize not enough for all tags", func() {
			tagsStr := DefaultTagsLimiter(tags, 8)

			So(tagsStr, ShouldResemble, " [tag1]")
		})

		Convey("with one long tag > maxSize", func() {
			tagsStr := DefaultTagsLimiter([]string{"long_tag"}, 4)

			So(tagsStr, ShouldResemble, "")
		})
	})
}
