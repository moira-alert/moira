package remote

import (
	"encoding/json"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPrepareRequest(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	Convey("Given valid params", t, func() {
		cfg := &Config{
			URL: "http://test/",
		}
		req, err := prepareRequest(from, until, target, cfg)
		Convey("url should be encoded correctly without error", func() {
			So(err, ShouldBeNil)
			So(req.URL.String(), ShouldEqual, "http://test/?format=json&from=300&target=foo.bar&until=500")
		})

		Convey("auth header should be empty", func() {
			So(req.Header.Get("Authorization"), ShouldEqual, "")
		})
	})
	Convey("Given valid params with user and password", t, func() {
		cfg := &Config{
			URL:      "http://test/",
			User:     "foo",
			Password: "bar",
		}
		req, err := prepareRequest(from, until, target, cfg)
		Convey("auth header should be set without error", func() {
			u, p, ok := req.BasicAuth()
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u, ShouldEqual, cfg.User)
			So(p, ShouldEqual, cfg.Password)
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

func TestConfig(t *testing.T) {
	Convey("Given config without url and enabled", t, func() {
		cfg := &Config{
			URL:     "",
			Enabled: true,
		}
		Convey("remote triggers should be disabled", func() {
			So(cfg.IsEnabled(), ShouldBeFalse)
		})
	})

	Convey("Given config with url and enabled", t, func() {
		cfg := &Config{
			URL:     "http://host",
			Enabled: true,
		}
		Convey("remote triggers should be enabled", func() {
			So(cfg.IsEnabled(), ShouldBeTrue)
		})
	})

	Convey("Given config with url and disabled", t, func() {
		cfg := &Config{
			URL:     "http://host",
			Enabled: false,
		}
		Convey("remote triggers should be disabled", func() {
			So(cfg.IsEnabled(), ShouldBeFalse)
		})
	})

	Convey("Given config without url and disabled", t, func() {
		cfg := &Config{
			URL:     "",
			Enabled: false,
		}
		Convey("remote triggers should be disabled", func() {
			So(cfg.IsEnabled(), ShouldBeFalse)
		})
	})
}

func TestConvertResponse(t *testing.T) {
	d := types.MakeMetricData("test", []float64{1, 2, 3}, 20, 0)
	data := []*types.MetricData{d}
	Convey("Given data and allowRealTimeAlerting is set", t, func() {
		ts := convertResponse(data, true)
		Convey("response should contain last value", func() {
			So(ts[0].MetricData.Values, ShouldResemble, []float64{1, 2, 3})
		})
	})
	Convey("Given data and allowRealTimeAlerting is not set", t, func() {
		ts := convertResponse(data, false)
		Convey("response should not contain last value", func() {
			So(ts[0].MetricData.Values, ShouldResemble, []float64{1, 2})
		})
	})
}
