package templating

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_TemplateDescription(t *testing.T) {
	Convey("Test templates", t, func() {
		triggerName := "TestName"
		template := "" +
			"Trigger name: {{.Trigger.Name}}\n" +
			"{{range $v := .Events }}\n" +
			"Metric: {{$v.Metric}}\n" +
			"MetricElements: {{$v.MetricElements}}\n" +
			"Timestamp: {{$v.Timestamp}}\n" +
			"Value: {{$v.Value}}\n" +
			"State: {{$v.State}}\n" +
			"{{end}}\n" +
			"https://grafana.yourhost.com/some-dashboard" +
			"{{ range $i, $v := .Events }}{{ if ne $i 0 }}&{{ else }}?" +
			"{{ end }}var-host={{ $v.Metric }}{{ end }}"

		testUnixTime := time.Now().Unix()
		events := []Event{
			{Metric: "1", Timestamp: testUnixTime},
			{Metric: "2", Timestamp: testUnixTime},
		}
		triggerDescriptionPopulater := NewTriggerDescriptionPopulater(triggerName, events)

		Convey("Test nil data", func() {
			triggerDescriptionPopulater = NewTriggerDescriptionPopulater(triggerName, nil)
			actual, err := triggerDescriptionPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, `Trigger name: TestName

https://grafana.yourhost.com/some-dashboard`)
		})

		Convey("Test data", func() {
			actual, err := triggerDescriptionPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, fmt.Sprintf("Trigger name: TestName\n\nMetric: 1\nMetricElements: []\nTimestamp: %d\nValue: &lt;nil&gt;"+
				"\nState: \n\nMetric: 2\nMetricElements: []\nTimestamp: %d\nValue: &lt;nil&gt;"+
				"\nState: \n\nhttps://grafana.yourhost.com/some-dashboard?var-host=1&var-host=2", testUnixTime, testUnixTime))
		})

		Convey("Test description without templates", func() {
			template = "Another text"

			actual, err := triggerDescriptionPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, template)
		})

		Convey("Test method Date", func() {
			formatDate := time.Unix(testUnixTime, 0).Format(eventTimeFormat)
			expected := fmt.Sprintf("%s | %s |", formatDate, formatDate)
			template = "{{ range .Events }}{{ date .Timestamp }} | {{ end }}"

			actual, err := triggerDescriptionPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, expected)
		})

		Convey("Test method formatted Date", func() {
			formatedDate := time.Unix(testUnixTime, 0).Format("2006-01-02 15:04:05")
			expected := fmt.Sprintf("%s | %s |", formatedDate, formatedDate)
			template = "{{ range .Events }}{{ formatDate .Timestamp \"2006-01-02 15:04:05\" }} | {{ end }}"

			actual, err := triggerDescriptionPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, expected)
		})

		Convey("Test method decrease and increase Date", func() {
			var timeOffset int64 = 300

			Convey("Date increase", func() {
				increase := testUnixTime + timeOffset
				expected := fmt.Sprintf("%d | %d |", increase, increase)
				template = fmt.Sprintf("{{ range .Events }}{{ .TimestampIncrease %d }} | {{ end }}", timeOffset)

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Date decrease", func() {
				increase := testUnixTime - timeOffset
				expected := fmt.Sprintf("%d | %d |", increase, increase)
				template = fmt.Sprintf("{{ range .Events }}{{ .TimestampDecrease %d }} | {{ end }}", timeOffset)

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})
		})

		Convey("Bad functions", func() {
			var timeOffset int64 = 300

			Convey("Non-existent function", func() {
				template = fmt.Sprintf("{{ range .Events }}{{ decrease %d }} | {{ end }}", timeOffset)

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldNotBeNil)
				So(actual, ShouldEqual, template)
			})

			Convey("Non-existent method", func() {
				template = fmt.Sprintf("{{ range .Events }}{{ .Decrease %d }} | {{ end }}", timeOffset)

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldNotBeNil)
				So(actual, ShouldEqual, template)
			})

			Convey("Bad parameters", func() {
				template = "{{ date \"bad\" }} "

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldNotBeNil)
				So(actual, ShouldEqual, template)
			})

			Convey("No parameters", func() {
				template = "{{ date }} "

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldNotBeNil)
				So(actual, ShouldEqual, template)
			})
		})

		Convey("Test strings functions", func() {
			Convey("Test replace", func() {
				template = "{{ stringsReplace \"my.metrics.path\" \".\" \"_\" -1 }} "
				expected := "my_metrics_path"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test replace limited to 1", func() {
				template = "{{ stringsReplace \"my.metrics.path\" \".\" \"_\" 1 }} "
				expected := "my_metrics.path"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test trim suffix", func() {
				template = "{{ stringsTrimSuffix \"my.metrics.path\" \".path\" }} "
				expected := "my.metrics"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test trim prefix", func() {
				template = "{{ stringsTrimPrefix \"my.metrics.path\" \"my.\" }} "
				expected := "metrics.path"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test lower case", func() {
				template = "{{ stringsToLower \"MY.PATH\" }} "
				expected := "my.path"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test upper case", func() {
				template = "{{ stringsToUpper \"my.path\" }} "
				expected := "MY.PATH"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})
		})

		Convey("Test some sprig functions", func() {
			Convey("Test upper", func() {
				template = "{{ \"hello!\" | upper}} "
				expected := "HELLO!"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test upper repeat", func() {
				template = "{{ \"hello!\" | upper | repeat 5 }} "
				expected := "HELLO!HELLO!HELLO!HELLO!HELLO!"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Test list uniq without", func() {
				template = "{{ without (list 1 3 3 2 2 2 4 4 4 4 1 | uniq) 4 }} "
				expected := "[1 3 2]"

				actual, err := triggerDescriptionPopulater.Populate(template)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})
		})
	})
}
