package templating

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/smartystreets/goconvey/convey"
)

type testPopulater struct{}

func (testData *testPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, testData)
}

func Test_populate(t *testing.T) {
	Convey("Test populate function", t, func() {
		populater := testPopulater{}

		Convey("Test strings functions", func() {
			Convey("Test replace", func() {
				template := "{{ stringsReplace \"my.metrics.path\" \".\" \"_\" -1 }} "
				expected := "my_metrics_path"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test replace limited to 1", func() {
				template := "{{ stringsReplace \"my.metrics.path\" \".\" \"_\" 1 }} "
				expected := "my_metrics.path"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test trim suffix", func() {
				template := "{{ stringsTrimSuffix \"my.metrics.path\" \".path\" }} "
				expected := "my.metrics"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test trim prefix", func() {
				template := "{{ stringsTrimPrefix \"my.metrics.path\" \"my.\" }} "
				expected := "metrics.path"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test lower case", func() {
				template := "{{ stringsToLower \"MY.PATH\" }} "
				expected := "my.path"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test upper case", func() {
				template := "{{ stringsToUpper \"my.path\" }} "
				expected := "MY.PATH"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})
		})

		Convey("Test some sprig functions", func() {
			Convey("Test upper", func() {
				template := "{{ \"hello!\" | upper}} "
				expected := "HELLO!"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test upper repeat", func() {
				template := "{{ \"hello!\" | upper | repeat 5 }} "
				expected := "HELLO!HELLO!HELLO!HELLO!HELLO!"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test list uniq without", func() {
				template := "{{ without (list 1 3 3 2 2 2 4 4 4 4 1 | uniq) 4 }} "
				expected := "[1 3 2]"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test uuidv4 function", func() {
				template := "{{ uuidv4 }}"

				actual, err := populater.Populate(template)
				So(err, ShouldBeNil)

				_, err = uuid.Parse(actual)
				So(err, ShouldBeNil)
			})
		})
	})
}
