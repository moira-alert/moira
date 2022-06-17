package remote

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	mock_clock "github.com/moira-alert/moira/mock/clock"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIsConfigured(t *testing.T) {
	Convey("Remote is not configured", t, func() {
		remote := Create(&Config{URL: "", Enabled: true})
		isConfigured, err := remote.IsConfigured()
		So(isConfigured, ShouldBeFalse)
		So(err, ShouldResemble, ErrRemoteStorageDisabled)
	})

	Convey("Remote is configured", t, func() {
		remote := Create(&Config{URL: "http://host", Enabled: true})
		isConfigured, err := remote.IsConfigured()
		So(isConfigured, ShouldBeTrue)
		So(err, ShouldBeEmpty)
	})
}

func TestIsRemoteAvailable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	systemClock := mock_clock.NewMockClock(mockCtrl)
	systemClock.EXPECT().Sleep(time.Second).Times(0)
	testConfigs := []Config{
		{},
		{HealthCheckRetrySeconds: []time.Duration{time.Second}},
		{HealthCheckRetrySeconds: []time.Duration{time.Second}},
		{HealthCheckRetrySeconds: []time.Duration{time.Second, time.Second}},
		{HealthCheckRetrySeconds: []time.Duration{time.Second, time.Second}},
		{HealthCheckRetrySeconds: []time.Duration{time.Second, time.Second, time.Second}},
	}
	body := []byte("Some string")

	Convey("Given server returns OK response the remote is available", t, func() {
		server := createServer(body, http.StatusOK)
		for _, config := range testConfigs {
			config.URL = server.URL
			remote := Remote{client: server.Client(), config: &config}
			isAvailable, err := remote.IsRemoteAvailable()
			So(isAvailable, ShouldBeTrue)
			So(err, ShouldBeEmpty)
		}
	})

	Convey("Given server returns Remote Unavailable responses permanently", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			server := createTestServer(TestResponse{body, statusCode})

			Convey(fmt.Sprintf(
				"request failed with %d response status code and remote is unavailable", statusCode,
			), func() {
				remote := Remote{client: server.Client()}
				for _, config := range testConfigs {
					config.URL = server.URL
					remote.config = &config
					systemClock := mock_clock.NewMockClock(mockCtrl)
					systemClock.EXPECT().Sleep(time.Second).Times(len(config.HealthCheckRetrySeconds))
					remote.clock = systemClock

					isAvailable, err := remote.IsRemoteAvailable()
					So(err, ShouldResemble, fmt.Errorf(
						"the remote server is not available. Response status %d: %s", statusCode, string(body),
					))
					So(isAvailable, ShouldBeFalse)
				}
			})
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"the remote is available with retry after %d response", statusCode,
			), func() {
				for _, config := range testConfigs {
					if len(config.HealthCheckRetrySeconds) == 0 {
						continue
					}
					server := createTestServer(
						TestResponse{body, statusCode},
						TestResponse{body, http.StatusOK},
					)
					config.URL = server.URL
					systemClock := mock_clock.NewMockClock(mockCtrl)
					systemClock.EXPECT().Sleep(time.Second).Times(1)
					remote := Remote{client: server.Client(), config: &config, clock: systemClock}

					isAvailable, err := remote.IsRemoteAvailable()
					So(err, ShouldBeNil)
					So(isAvailable, ShouldBeTrue)
				}
			})
		}
	})
}

func TestFetch(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar" //nolint
	testConfigs := []Config{
		{},
		{RetrySeconds: []time.Duration{time.Second}},
		{RetrySeconds: []time.Duration{time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second, time.Second}},
	}
	mockCtrl := gomock.NewController(t)
	validBody := []byte("[{\"Target\": \"t1\",\"DataPoints\":[[1,2],[3,4]]}]")

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
		_, ok := err.(ErrRemoteTriggerResponse)
		So(ok, ShouldBeTrue)
	})

	Convey("Fail request with InternalServerError", t, func() {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		remote := Remote{client: server.Client()}
		for _, config := range testConfigs {
			config.URL = server.URL
			remote.config = &config
			result, err := remote.Fetch(target, from, until, false)
			So(result, ShouldBeEmpty)
			So(err.Error(), ShouldResemble, fmt.Sprintf("remote server response status %d: %s", http.StatusInternalServerError, "Some string"))
			_, ok := err.(ErrRemoteTriggerResponse)
			So(ok, ShouldBeTrue)
		}
	})

	Convey("Client calls bad url", t, func() {
		url := "💩%$&TR"
		for _, config := range testConfigs {
			config.URL = url
			remote := Remote{config: &config}
			result, err := remote.Fetch(target, from, until, false)
			So(result, ShouldBeEmpty)
			So(err.Error(), ShouldResemble, "parse \"💩%$&TR\": invalid URL escape \"%$&\"")
			_, ok := err.(ErrRemoteTriggerResponse)
			So(ok, ShouldBeTrue)
		}
	})

	Convey("Given server returns Remote Unavailable responses permanently", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			server := createTestServer(TestResponse{validBody, statusCode})

			Convey(fmt.Sprintf(
				"request failed with %d response status code and remote is unavailable", statusCode,
			), func() {
				remote := Remote{client: server.Client()}
				for _, config := range testConfigs {
					config.URL = server.URL
					remote.config = &config
					systemClock := mock_clock.NewMockClock(mockCtrl)
					systemClock.EXPECT().Sleep(time.Second).Times(len(config.RetrySeconds))
					remote.clock = systemClock

					result, err := remote.Fetch(target, from, until, false)
					So(err, ShouldResemble, ErrRemoteUnavailable{
						InternalError: fmt.Errorf(
							"the remote server is not available. Response status %d: %s", statusCode, string(validBody),
						), Target: target,
					})
					So(result, ShouldBeNil)
				}
			})
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"the remote is available with retry after %d response", statusCode,
			), func() {
				for _, config := range testConfigs {
					if len(config.RetrySeconds) == 0 {
						continue
					}
					server := createTestServer(
						TestResponse{validBody, statusCode},
						TestResponse{validBody, http.StatusOK},
					)
					config.URL = server.URL
					systemClock := mock_clock.NewMockClock(mockCtrl)
					systemClock.EXPECT().Sleep(time.Second).Times(1)
					remote := Remote{client: server.Client(), config: &config, clock: systemClock}

					result, err := remote.Fetch(target, from, until, false)
					So(err, ShouldBeNil)
					So(result, ShouldNotBeNil)
					metricsData := result.GetMetricsData()
					So(len(metricsData), ShouldEqual, 1)
					So(metricsData[0].Name, ShouldEqual, "t1")
				}
			})
		}
	})
}
