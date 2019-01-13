package remote

import (
	"testing"

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
