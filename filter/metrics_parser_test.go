package filter

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParseMetric(t *testing.T) {
	type ValidMetricCase struct {
		input     string
		metric    string
		name      string
		labels    map[string]string
		value     float64
		timestamp int64
	}

	Convey("Given invalid metric strings, should return errors", t, func() {
		invalidMetrics := []string{
			"Invalid.value 12g5 1234567890",
			"No.value.two.spaces  1234567890",
			"No.timestamp.space.in.the.end 12 ",
			"No.value.no.timestamp",
			"No.timestamp 12",
			" 12 1234567890",
			"Non-ascii.こんにちは 12 1234567890",
			"Non-printable.\000 12 1234567890",
			"",
			"\n",
			"Too.many.parts 1 2 3 4 12 1234567890",
			"Space.in.the.end 12 1234567890 ",
			" Space.in.the.beginning 12 1234567890",
			"\tNon-printable.in.the.beginning 12 1234567890",
			"\rNon-printable.in.the.beginning 12 1234567890",
			"Newline.in.the.end 12 1234567890\n",
			"Newline.in.the.end 12 1234567890\r",
			"Newline.in.the.end 12 1234567890\r\n",
			";Empty.name.but.with.label= 1 2",
			"No.labels.but.delimiter.in.the.end; 1 2",
			"Empty.label.name;= 1 2",
			"Only.label.name;name 1 2",
		}

		for _, invalidMetric := range invalidMetrics {
			_, err := ParseMetric([]byte(invalidMetric))
			So(err, ShouldBeError)
		}
	})

	Convey("Given valid metric strings, should return parsed values", t, func() {
		validMetrics := []ValidMetricCase{
			{"One.two.three 123 1234567890", "One.two.three", "One.two.three", map[string]string{}, 123, 1234567890},
			{"One.two.three 1.23e2 1234567890", "One.two.three", "One.two.three", map[string]string{}, 123, 1234567890},
			{"One.two.three -123 1234567890", "One.two.three", "One.two.three", map[string]string{}, -123, 1234567890},
			{"One.two.three +123 1234567890", "One.two.three", "One.two.three", map[string]string{}, 123, 1234567890},
			{"One.two.three 123. 1234567890", "One.two.three", "One.two.three", map[string]string{}, 123, 1234567890},
			{"One.two.three 123.0 1234567890", "One.two.three", "One.two.three", map[string]string{}, 123, 1234567890},
			{"One.two.three .123 1234567890", "One.two.three", "One.two.three", map[string]string{}, 0.123, 1234567890},
			{"One.two.three;four=five 123 1234567890", "One.two.three;four=five", "One.two.three", map[string]string{"four": "five"}, 123, 1234567890},
			{"One.two.three;four= 123 1234567890", "One.two.three;four=", "One.two.three", map[string]string{"four": ""}, 123, 1234567890},
			{"One.two.three;six=seven;four=five 123 1234567890", "One.two.three;four=five;six=seven", "One.two.three", map[string]string{"four": "five", "six": "seven"}, 123, 1234567890},
			{"One.two.three;four=five;six=seven=eight 123 1234567890", "One.two.three;four=five;six=seven=eight", "One.two.three", map[string]string{"four": "five", "six": "seven=eight"}, 123, 1234567890},
			{"One.two.three;four=five;six=seven=eight=nine 123 1234567890", "One.two.three;four=five;six=seven=eight=nine",
				"One.two.three", map[string]string{"four": "five", "six": "seven=eight=nine"}, 123, 1234567890},
			{"One.two.three;four=five;six=seven=eight=nine= 123 1234567890", "One.two.three;four=five;six=seven=eight=nine=",
				"One.two.three", map[string]string{"four": "five", "six": "seven=eight=nine="}, 123, 1234567890},
		}

		for _, validMetric := range validMetrics {
			parsedMetric, err := ParseMetric([]byte(validMetric.input))
			So(err, ShouldBeEmpty)
			So(parsedMetric.Metric, ShouldEqual, validMetric.metric)
			So(parsedMetric.Name, ShouldEqual, validMetric.name)
			So(parsedMetric.Labels, ShouldResemble, validMetric.labels)
			So(parsedMetric.Value, ShouldEqual, validMetric.value)
			So(parsedMetric.Timestamp, ShouldEqual, validMetric.timestamp)
		}

		Convey("Check metrics with magic '-1' timestamp", func() {
			timeStart := time.Now().Unix()
			magicTimestampMetrics := []ValidMetricCase{
				{
					input:     "One.two.three 123 -1",
					metric:    "One.two.three",
					name:      "Metric with integer value and magic -1 as time value",
					labels:    map[string]string{},
					value:     123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three 1.23e2 -1",
					metric:    "One.two.three",
					name:      "Metric with float value and magic -1 as time value",
					labels:    map[string]string{},
					value:     123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three -123 -1",
					metric:    "One.two.three",
					name:      "Metric with negative integer value and magic -1 as time value",
					labels:    map[string]string{},
					value:     -123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three +123 -1",
					metric:    "One.two.three",
					name:      "Metric with positive integer value and magic -1 as time value",
					labels:    map[string]string{},
					value:     123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three 123. -1",
					metric:    "One.two.three",
					name:      "Metric with integer with point value and magic -1 as time value",
					labels:    map[string]string{},
					value:     123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three 123.0 -1",
					metric:    "One.two.three",
					name:      "Metric with integer with 0 value and magic -1 as time value",
					labels:    map[string]string{},
					value:     123,
					timestamp: timeStart,
				},
				{
					input:     "One.two.three .123 -1",
					metric:    "One.two.three",
					name:      "Metric with float without 0 value and magic -1 as time value",
					labels:    map[string]string{},
					value:     0.123,
					timestamp: timeStart,
				},
			}

			for _, magicMetric := range magicTimestampMetrics {
				Convey(magicMetric.name, func() {
					parsedMetric, err := ParseMetric([]byte(magicMetric.input))
					So(err, ShouldBeEmpty)
					So(parsedMetric.Metric, ShouldEqual, magicMetric.metric)
					So(parsedMetric.Labels, ShouldResemble, magicMetric.labels)
					So(parsedMetric.Value, ShouldEqual, magicMetric.value)
					// I add 5 seconds to avoid false failures
					So(parsedMetric.Timestamp, ShouldBeBetweenOrEqual, magicMetric.timestamp, magicMetric.timestamp+5)
				})
			}
		})
	})

	Convey("Given valid metric strings with float64 timestamp, should return parsed values", t, func() {
		var testTimestamp int64 = 1234567890

		// Create and test n metrics with float64 timestamp with fractional part of length n (n=19)
		//
		// For example:
		//
		// [n=1] One.two.three 123 1234567890.6
		// [n=2] One.two.three 123 1234567890.94
		// [n=3] One.two.three 123 1234567890.665
		// [n=4] One.two.three 123 1234567890.4377
		// ...
		// [n=19] One.two.three 123 1234567890.6790847778320312500

		for i := 1; i < 20; i++ {
			rawTimestamp := strconv.FormatFloat(float64(testTimestamp)+rand.Float64(), 'f', i, 64)
			rawMetric := "One.two.three 123 " + rawTimestamp
			validMetric := ValidMetricCase{rawMetric, "One.two.three", "One.two.three", map[string]string{}, 123, testTimestamp}
			parsedMetric, err := ParseMetric([]byte(validMetric.input))
			So(err, ShouldBeEmpty)
			So(parsedMetric.Metric, ShouldResemble, validMetric.metric)
			So(parsedMetric.Name, ShouldResemble, validMetric.name)
			So(parsedMetric.Labels, ShouldResemble, validMetric.labels)
			So(parsedMetric.Value, ShouldEqual, validMetric.value)
			So(parsedMetric.Timestamp, ShouldEqual, validMetric.timestamp)
		}
	})
}

func TestRestoreMetricStringByNameAndLabels(t *testing.T) {
	Convey("Test function restoreMetricStringByNameAndLabels", t, func() {
		Convey("Given two metrics with the same labels but in a different order", func() {
			testCases := []struct {
				name   string
				labels map[string]string
			}{
				{"One.two.three", map[string]string{"one": "two", "four": "five", "six": "seven"}},
				{"One.two.three", map[string]string{"six": "seven", "four": "five", "one": "two"}},
			}
			expected := "One.two.three;four=five;one=two;six=seven"
			Convey("Result of restored metric should be equal", func() {
				for _, testCase := range testCases {
					actual := restoreMetricStringByNameAndLabels(testCase.name, testCase.labels)
					So(actual, ShouldEqual, expected)
				}
			})
		})
	})
}

func TestParsedMetric_IsTooOld(t *testing.T) {
	now := time.Date(2022, 6, 16, 10, 0, 0, 0, time.UTC)
	maxTTL := time.Hour

	Convey("When metric is old, return true", t, func() {
		metric := ParsedMetric{
			Name:      "too old metric",
			Timestamp: time.Date(2022, 6, 16, 8, 59, 0, 0, time.UTC).Unix(),
		}
		So(metric.IsTooOld(maxTTL, now), ShouldBeTrue)
	})
	Convey("When metric is young, return false", t, func() {
		metric := ParsedMetric{
			Name:      "too old metric",
			Timestamp: time.Date(2022, 6, 16, 9, 00, 0, 0, time.UTC).Unix(),
		}
		So(metric.IsTooOld(maxTTL, now), ShouldBeFalse)
	})
}
