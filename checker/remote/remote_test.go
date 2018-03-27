package remote

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPrepareRequest(t *testing.T) {
	Convey("Given valid params", t, func() {
		url := "http://test/"
		var from int64 = 300
		var until int64 = 500
		target := "foo.bar"
		req, err := prepareRequest(url, from, until, target)
		Convey("url should be encoded correctly and without error", func() {
			So(err, ShouldBeNil)
			So(req.URL.String(), ShouldEqual, "http://test/?format=json&from=300&target=foo.bar&until=500")
		})
	})
}

func TestDecodeBody(t *testing.T) {
	Convey("Given empty json response", t, func() {
		body := []byte("[]")
		resp, err := decodeBody(body)
		Convey("response should be empty and without error", func() {
			So(err, ShouldBeNil)
			So(len(resp), ShouldEqual, 0)
		})
	})

	Convey("Given response with only null points", t, func() {
		f := 1522076567.0
		f2 := 1522076867.0
		r := []graphiteMetric{{Target: "t", Datapoints: [][2]*float64{
			{nil, &f},
			{nil, &f2},
		}}}
		body, _ := json.Marshal(r)

		resp, err := decodeBody(body)
		Convey("length should be one", func() {
			So(resp, ShouldHaveLength, 1)
		})
		Convey("response should not contain any Values", func() {
			So(err, ShouldBeNil)
			for _, v := range resp[0].FetchResponse.IsAbsent {
				So(v, ShouldBeTrue)
			}
		})
		Convey("step size should be default", func() {
			So(resp[0].FetchResponse.StepTime, ShouldEqual, int32(f2-f))
		})
	})

	Convey("Given response with only last not null point", t, func() {
		f := 1522076567.0
		f2 := 1522076867.0
		p1 := 233.0
		r := []graphiteMetric{{Target: "t", Datapoints: [][2]*float64{
			{nil, &f},
			{&p1, &f2},
		}}}
		body, _ := json.Marshal(r)

		resp, err := decodeBody(body)
		Convey("second response value should be set", func() {
			So(err, ShouldBeNil)
			fr := resp[0].FetchResponse
			So(fr.IsAbsent[0], ShouldBeTrue)
			So(fr.IsAbsent[1], ShouldBeFalse)
			So(fr.Values[1], ShouldEqual, p1)
		})
	})
}
