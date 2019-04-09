package remote

import (
	"encoding/json"
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDecodeBody(t *testing.T) {
	Convey("Given empty json response", t, func(c C) {
		body := []byte("[]")
		resp, err := decodeBody(body)
		Convey("response should be empty and without error", t, func(c C) {
			c.So(err, ShouldBeNil)
			c.So(len(resp), ShouldEqual, 0)
		})
	})

	Convey("Given response with only null points", t, func(c C) {
		f := 1522076567.0
		f2 := 1522076867.0
		r := []graphiteMetric{{Target: "t", DataPoints: [][2]*float64{
			{nil, &f},
			{nil, &f2},
		}}}
		body, _ := json.Marshal(r)

		resp, err := decodeBody(body)
		Convey("length should be one", t, func(c C) {
			c.So(resp, ShouldHaveLength, 1)
		})
		Convey("response should not contain any Values", t, func(c C) {
			c.So(err, ShouldBeNil)
		})
		Convey("step size should be default", t, func(c C) {
			c.So(resp[0].StepTime, ShouldEqual, int32(f2-f))
		})
	})

	Convey("Given response with only last not null point", t, func(c C) {
		f := 1522076567.0
		f2 := 1522076867.0
		p1 := 233.0
		r := []graphiteMetric{{Target: "t", DataPoints: [][2]*float64{
			{nil, &f},
			{&p1, &f2},
		}}}
		body, _ := json.Marshal(r)

		resp, err := decodeBody(body)
		Convey("second response value should be set", t, func(c C) {
			c.So(err, ShouldBeNil)
			fr := resp[0]
			c.So(fr.Values[1], ShouldEqual, p1)
		})
	})
}

func TestConvertResponse(t *testing.T) {
	d := metricSource.MakeMetricData("test", []float64{1, 2, 3}, 20, 0)
	data := []*metricSource.MetricData{d}
	Convey("Given data and allowRealTimeAlerting is set", t, func(c C) {
		fetchResult := convertResponse(data, true)
		Convey("response should contain last value", t, func(c C) {
			c.So(fetchResult.MetricsData[0].Values, ShouldResemble, []float64{1, 2, 3})
		})
	})
	Convey("Given data and allowRealTimeAlerting is not set", t, func(c C) {
		fetchResult := convertResponse(data, false)
		Convey("response should not contain last value", t, func(c C) {
			c.So(fetchResult.MetricsData[0].Values, ShouldResemble, []float64{1, 2})
		})
	})
}
