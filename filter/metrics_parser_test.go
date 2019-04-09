package filter

import (
	"math/rand"
	"strconv"
	"testing"

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

	Convey("Given invalid metric strings, should return errors", t, func(c C) {
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
			"Too.Many.=.In.Label;name=value= 1 2",
		}

		for _, invalidMetric := range invalidMetrics {
			_, err := ParseMetric([]byte(invalidMetric))
			c.So(err, ShouldBeError)
		}
	})

	Convey("Given valid metric strings, should return parsed values", t, func(c C) {
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
			{"One.two.three;four=five;six=seven 123 1234567890", "One.two.three;four=five;six=seven", "One.two.three", map[string]string{"four": "five", "six": "seven"}, 123, 1234567890},
		}

		for _, validMetric := range validMetrics {
			parsedMetric, err := ParseMetric([]byte(validMetric.input))
			c.So(err, ShouldBeEmpty)
			c.So(parsedMetric.Metric, ShouldEqual, validMetric.metric)
			c.So(parsedMetric.Name, ShouldEqual, validMetric.name)
			c.So(parsedMetric.Labels, ShouldResemble, validMetric.labels)
			c.So(parsedMetric.Value, ShouldEqual, validMetric.value)
			c.So(parsedMetric.Timestamp, ShouldEqual, validMetric.timestamp)
		}
	})

	Convey("Given valid metric strings with float64 timestamp, should return parsed values", t, func(c C) {
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
			c.So(err, ShouldBeEmpty)
			c.So(parsedMetric.Metric, ShouldResemble, validMetric.metric)
			c.So(parsedMetric.Name, ShouldResemble, validMetric.name)
			c.So(parsedMetric.Labels, ShouldResemble, validMetric.labels)
			c.So(parsedMetric.Value, ShouldEqual, validMetric.value)
			c.So(parsedMetric.Timestamp, ShouldEqual, validMetric.timestamp)
		}
	})
}
