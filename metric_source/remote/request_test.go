package remote

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPrepareRequest(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	Convey("Given valid params", t, func() {
		remote := Remote{config: &Config{
			URL: "http://test/",
		}}
		req, err := remote.prepareRequest(from, until, target)
		Convey("url should be encoded correctly without error", func() {
			So(err, ShouldBeNil)
			So(req.URL.String(), ShouldEqual, "http://test/?format=json&from=300&target=foo.bar&until=500")
		})

		Convey("auth header should be empty", func() {
			So(req.Header.Get("Authorization"), ShouldEqual, "")
		})
	})
	Convey("Given valid params with user and password", t, func() {
		remote := Remote{config: &Config{
			URL:      "http://test/",
			User:     "foo",
			Password: "bar",
		}}
		req, err := remote.prepareRequest(from, until, target)
		Convey("auth header should be set without error", func() {
			u, p, ok := req.BasicAuth()
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u, ShouldEqual, remote.config.User)
			So(p, ShouldEqual, remote.config.Password)
		})
	})
}

func TestMakeRequest(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	body := []byte("Some string")

	Convey("Client returns status OK", t, func() {
		server := createServer(body, http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		request, _ := remote.prepareRequest(from, until, target)
		actual, err := remote.makeRequest(request)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, body)
	})

	Convey("Client returns status InternalServerError", t, func() {
		server := createServer(body, http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		request, _ := remote.prepareRequest(from, until, target)
		actual, err := remote.makeRequest(request)
		So(err, ShouldResemble, fmt.Errorf("bad response status %d: %s", http.StatusInternalServerError, string(body)))
		So(actual, ShouldResemble, body)
	})

	Convey("Client calls bad url", t, func() {
		server := createServer(body, http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: "http://bad/"}}
		request, _ := remote.prepareRequest(from, until, target)
		actual, err := remote.makeRequest(request)
		So(err, ShouldNotBeEmpty)
		So(actual, ShouldBeEmpty)
	})
}

func createServer(body []byte, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(statusCode)
		rw.Write(body)
	}))
}
