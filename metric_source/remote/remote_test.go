package remote

import (
	"fmt"
	"net/http"
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIsRemoteAvailable(t *testing.T) {
	Convey("Is available", t, func() {
		server := createServer([]byte("Some string"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		isAvailable, err := remote.IsAvailable()
		So(isAvailable, ShouldBeTrue)
		So(err, ShouldBeEmpty)
	})

	Convey("Not available", t, func() {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		isAvailable, err := remote.IsAvailable()
		So(isAvailable, ShouldBeFalse)
		So(err, ShouldResemble, fmt.Errorf("bad response status %d: %s", http.StatusInternalServerError, "Some string"))
	})
}

func TestFetch(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar" //nolint

	Convey("Request success but body is invalid", t, func() {
		server := createServer([]byte("[]"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldResemble, &FetchResult{MetricsData: []metricSource.MetricData{}})
		So(err, ShouldBeEmpty)
	})

	Convey("Request success but body is invalid", t, func() {
		server := createServer([]byte("Some string"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldBeEmpty)
		So(err.Error(), ShouldResemble, "invalid character 'S' looking for beginning of value")
	})

	Convey("Fail request with InternalServerError", t, func() {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldBeEmpty)
		So(err.Error(), ShouldResemble, fmt.Sprintf("bad response status %d: %s", http.StatusInternalServerError, "Some string"))
	})

	Convey("Fail make request", t, func() {
		url := "💩%$&TR"
		remote := Remote{config: &Config{URL: url}}
		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldBeEmpty)
		So(err.Error(), ShouldResemble, "parse \"💩%$&TR\": invalid URL escape \"%$&\"")
	})
}
