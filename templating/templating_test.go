package templating

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_TemplateDescription(t *testing.T) {
	Convey("Test templates", t, func() {
		var Name = "TestName"
		var Desc = "" +
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

		var testUnixTime = time.Now().Unix()
		var events = []Event{{Metric: "1", Timestamp: testUnixTime}, {Metric: "2", Timestamp: testUnixTime}}

		Convey("Test nil data", func() {
			expected, err := Populate(Name, Desc, nil)
			if err != nil {
				println("Error:", err.Error())
			}
			So(err, ShouldBeNil)
			So(`Trigger name: TestName

https://grafana.yourhost.com/some-dashboard`,
				ShouldResemble, expected)
		})

		Convey("Test data", func() {
			expected, err := Populate(Name, Desc, events)
			So(err, ShouldBeNil)
			So(fmt.Sprintf("Trigger name: TestName\n\nMetric: 1\nMetricElements: []\nTimestamp: %d\nValue: &lt;nil&gt;"+
				"\nState: \n\nMetric: 2\nMetricElements: []\nTimestamp: %d\nValue: &lt;nil&gt;"+
				"\nState: \n\nhttps://grafana.yourhost.com/some-dashboard?var-host=1&var-host=2", testUnixTime, testUnixTime),
				ShouldResemble, expected)
		})

		Convey("Test description without templates", func() {
			anotherText := "Another text"
			Desc = anotherText

			expected, err := Populate(Name, Desc, events)
			So(err, ShouldBeNil)
			So(anotherText, ShouldEqual, expected)
		})

		Convey("Test method Date", func() {
			formatDate := time.Unix(testUnixTime, 0).Format(eventTimeFormat)
			actual := fmt.Sprintf("%s | %s |", formatDate, formatDate)
			Desc = "{{ range .Events }}{{ date .Timestamp }} | {{ end }}"

			expected, err := Populate(Name, Desc, events)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, expected)
		})

		Convey("Test method formatted Date", func() {
			formatedDate := time.Unix(testUnixTime, 0).Format("2006-01-02 15:04:05")
			actual := fmt.Sprintf("%s | %s |", formatedDate, formatedDate)
			Desc = "{{ range .Events }}{{ formatDate .Timestamp \"2006-01-02 15:04:05\" }} | {{ end }}"

			expected, err := Populate(Name, Desc, events)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, expected)
		})

		Convey("Test method decrease and increase Date", func() {
			var timeOffset int64 = 300

			Convey("Date increase", func() {
				increase := testUnixTime + timeOffset
				actual := fmt.Sprintf("%d | %d |", increase, increase)
				Desc = fmt.Sprintf("{{ range .Events }}{{ .TimestampIncrease %d }} | {{ end }}", timeOffset)

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})

			Convey("Date decrease", func() {
				increase := testUnixTime - timeOffset
				actual := fmt.Sprintf("%d | %d |", increase, increase)
				Desc = fmt.Sprintf("{{ range .Events }}{{ .TimestampDecrease %d }} | {{ end }}", timeOffset)

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, expected)
			})
		})

		Convey("Bad functions", func() {
			var timeOffset int64 = 300

			Convey("Non-existent function", func() {
				Desc = fmt.Sprintf("{{ range .Events }}{{ decrease %d }} | {{ end }}", timeOffset)

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldNotBeNil)
				So(Desc, ShouldEqual, expected)
			})

			Convey("Non-existent method", func() {
				Desc = fmt.Sprintf("{{ range .Events }}{{ .Decrease %d }} | {{ end }}", timeOffset)

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldNotBeNil)
				So(Desc, ShouldEqual, expected)
			})

			Convey("Bad parameters", func() {
				Desc = "{{ date \"bad\" }} "

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldNotBeNil)
				So(Desc, ShouldEqual, expected)
			})

			Convey("No parameters", func() {
				Desc = "{{ date }} "

				expected, err := Populate(Name, Desc, events)
				So(err, ShouldNotBeNil)
				So(Desc, ShouldEqual, expected)
			})
		})
	})
}
