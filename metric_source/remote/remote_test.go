package remote

import (
	"fmt"
	"net/http"
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIsConfigured(t *testing.T) {
	Convey("Remote is not configured", t, func(c C) {
		remote := Create(&Config{URL: "", Enabled: true})
		isConfigured, err := remote.IsConfigured()
		c.So(isConfigured, ShouldBeFalse)
		c.So(err, ShouldResemble, ErrRemoteStorageDisabled)
	})

	Convey("Remote is configured", t, func(c C) {
		remote := Create(&Config{URL: "http://host", Enabled: true})
		isConfigured, err := remote.IsConfigured()
		c.So(isConfigured, ShouldBeTrue)
		c.So(err, ShouldBeEmpty)
	})
}

func TestIsRemoteAvailable(t *testing.T) {
	Convey("Is available", t, func(c C) {
		server := createServer([]byte("Some string"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		isAvailable, err := remote.IsRemoteAvailable()
		c.So(isAvailable, ShouldBeTrue)
		c.So(err, ShouldBeEmpty)
	})

	Convey("Not available", t, func(c C) {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		isAvailable, err := remote.IsRemoteAvailable()
		c.So(isAvailable, ShouldBeFalse)
		c.So(err, ShouldResemble, fmt.Errorf("bad response status %d: %s", http.StatusInternalServerError, "Some string"))
	})
}

func TestFetch(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"

	Convey("Request success but body is invalid", t, func(c C) {
		server := createServer([]byte("[]"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		c.So(result, ShouldResemble, &FetchResult{MetricsData: []*metricSource.MetricData{}})
		c.So(err, ShouldBeEmpty)
	})

	Convey("Request success but body is invalid", t, func(c C) {
		server := createServer([]byte("Some string"), http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		c.So(result, ShouldBeEmpty)
		c.So(err.Error(), ShouldResemble, "failed to get remote target 'foo.bar': invalid character 'S' looking for beginning of value")
	})

	Convey("Fail request with InternalServerError", t, func(c C) {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		result, err := remote.Fetch(target, from, until, false)
		c.So(result, ShouldBeEmpty)
		c.So(err.Error(), ShouldResemble, fmt.Sprintf("failed to get remote target 'foo.bar': bad response status %d: %s", http.StatusInternalServerError, "Some string"))
	})

	Convey("Fail make request", t, func(c C) {
		url := "💩%$&TR"
		remote := Remote{config: &Config{URL: url}}
		result, err := remote.Fetch(target, from, until, false)
		c.So(result, ShouldBeEmpty)
		c.So(err.Error(), ShouldResemble, "failed to get remote target 'foo.bar': parse 💩%$&TR: invalid URL escape \"%$&\"")
	})
}
