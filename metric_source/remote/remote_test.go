package remote

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/retries"

	. "github.com/smartystreets/goconvey/convey"
)

var testConfigs = []*Config{
	{
		Timeout: time.Second,
		Retries: retries.Config{
			InitialInterval:     time.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2,
			MaxInterval:         time.Millisecond * 20,
			MaxRetriesCount:     2,
		},
	},
	{
		Timeout: time.Millisecond * 200,
		Retries: retries.Config{
			InitialInterval:     time.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2,
			MaxInterval:         time.Second,
			MaxElapsedTime:      time.Second * 2,
		},
	},
}

func TestIsAvailable(t *testing.T) {
	body := []byte("Some string")

	isAvailableTestConfigs := make([]*Config, 0, len(testConfigs))
	for _, conf := range testConfigs {
		isAvailableTestConfigs = append(isAvailableTestConfigs, &Config{
			HealthcheckTimeout: conf.Timeout,
			HealthcheckRetries: conf.Retries,
		})
	}

	retrier := retries.NewStandardRetrier[[]byte]()

	Convey("Given server returns OK response the remote is available", t, func() {
		server := createServer(body, http.StatusOK)
		defer server.Close()

		for _, config := range isAvailableTestConfigs {
			config.URL = server.URL

			remote := Remote{
				client:                    server.Client(),
				config:                    config,
				retrier:                   retrier,
				healthcheckBackoffFactory: retries.NewExponentialBackoffFactory(config.HealthcheckRetries),
			}

			isAvailable, err := remote.IsAvailable()
			So(isAvailable, ShouldBeTrue)
			So(err, ShouldBeEmpty)
		}
	})

	Convey("Given server returns Remote Unavailable responses permanently", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			server := createTestServer(TestResponse{body, statusCode})

			Convey(fmt.Sprintf(
				"request failed with %d response status code and remote is unavailable", statusCode,
			), func() {
				for _, config := range isAvailableTestConfigs {
					config.URL = server.URL

					remote := Remote{
						client:                    server.Client(),
						config:                    config,
						retrier:                   retrier,
						healthcheckBackoffFactory: retries.NewExponentialBackoffFactory(config.HealthcheckRetries),
					}

					isAvailable, err := remote.IsAvailable()
					So(err, ShouldResemble, ErrRemoteUnavailable{
						InternalError: fmt.Errorf(
							"the remote server is not available. Response status %d: %s", statusCode, string(body),
						),
					})
					So(isAvailable, ShouldBeFalse)
				}
			})

			server.Close()
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"the remote is available with retry after %d response", statusCode,
			), func() {
				for _, config := range isAvailableTestConfigs {
					server := createTestServer(
						TestResponse{body, statusCode},
						TestResponse{body, http.StatusOK},
					)
					config.URL = server.URL

					remote := Remote{
						client:                    server.Client(),
						config:                    config,
						retrier:                   retrier,
						healthcheckBackoffFactory: retries.NewExponentialBackoffFactory(config.HealthcheckRetries),
					}

					isAvailable, err := remote.IsAvailable()
					So(err, ShouldBeNil)
					So(isAvailable, ShouldBeTrue)

					server.Close()
				}
			})
		}
	})
}

func TestFetch(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar" //nolint

	retrier := retries.NewStandardRetrier[[]byte]()
	validBody := []byte("[{\"Target\": \"t1\",\"DataPoints\":[[1,2],[3,4]]}]")

	Convey("Request success but body is invalid", t, func() {
		server := createServer([]byte("[]"), http.StatusOK)

		conf := testConfigs[0]
		conf.URL = server.URL

		remote := Remote{
			client:                server.Client(),
			config:                conf,
			retrier:               retrier,
			requestBackoffFactory: retries.NewExponentialBackoffFactory(conf.Retries),
		}

		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldResemble, &FetchResult{MetricsData: []metricSource.MetricData{}})
		So(err, ShouldBeEmpty)
	})

	Convey("Request success but body is invalid", t, func() {
		server := createServer([]byte("Some string"), http.StatusOK)
		defer server.Close()

		conf := testConfigs[0]
		conf.URL = server.URL

		remote := Remote{
			client:                server.Client(),
			config:                conf,
			retrier:               retrier,
			requestBackoffFactory: retries.NewExponentialBackoffFactory(conf.Retries),
		}

		result, err := remote.Fetch(target, from, until, false)
		So(result, ShouldBeEmpty)
		So(err.Error(), ShouldResemble, "invalid character 'S' looking for beginning of value")
		So(err, ShouldHaveSameTypeAs, ErrRemoteTriggerResponse{})
	})

	Convey("Fail request with InternalServerError", t, func() {
		server := createServer([]byte("Some string"), http.StatusInternalServerError)
		defer server.Close()

		for _, config := range testConfigs {
			config.URL = server.URL

			remote := Remote{
				client:                server.Client(),
				config:                config,
				retrier:               retrier,
				requestBackoffFactory: retries.NewExponentialBackoffFactory(config.Retries),
			}

			result, err := remote.Fetch(target, from, until, false)

			So(result, ShouldBeEmpty)
			So(err.Error(), ShouldResemble, fmt.Sprintf("bad response status %d: %s", http.StatusInternalServerError, "Some string"))
			So(err, ShouldHaveSameTypeAs, ErrRemoteTriggerResponse{})
		}
	})

	Convey("Client calls bad url", t, func() {
		server := createTestServer(TestResponse{[]byte("Some string"), http.StatusOK})
		defer server.Close()

		url := "💩%$&TR"

		for _, config := range testConfigs {
			config.URL = url

			remote := Remote{
				client:                server.Client(),
				config:                config,
				retrier:               retrier,
				requestBackoffFactory: retries.NewExponentialBackoffFactory(config.Retries),
			}

			result, err := remote.Fetch(target, from, until, false)
			So(result, ShouldBeEmpty)
			So(err.Error(), ShouldResemble, "parse \"💩%$&TR\": invalid URL escape \"%$&\"")
			So(err, ShouldHaveSameTypeAs, ErrRemoteTriggerResponse{})
		}
	})

	Convey("Given server returns Remote Unavailable responses permanently", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			server := createTestServer(TestResponse{validBody, statusCode})

			Convey(fmt.Sprintf(
				"request failed with %d response status code and remote is unavailable", statusCode,
			), func() {
				for _, config := range testConfigs {
					config.URL = server.URL
					remote := Remote{
						client:                server.Client(),
						config:                config,
						retrier:               retrier,
						requestBackoffFactory: retries.NewExponentialBackoffFactory(config.Retries),
					}

					result, err := remote.Fetch(target, from, until, false)
					So(err, ShouldResemble, ErrRemoteUnavailable{
						InternalError: fmt.Errorf(
							"the remote server is not available. Response status %d: %s", statusCode, string(validBody),
						), Target: target,
					})
					So(result, ShouldBeNil)
				}
			})

			server.Close()
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"the remote is available with retry after %d response", statusCode,
			), func() {
				for _, config := range testConfigs {
					server := createTestServer(
						TestResponse{validBody, statusCode},
						TestResponse{validBody, http.StatusOK},
					)
					config.URL = server.URL

					remote := Remote{
						client:                server.Client(),
						config:                config,
						retrier:               retrier,
						requestBackoffFactory: retries.NewExponentialBackoffFactory(config.Retries),
					}

					result, err := remote.Fetch(target, from, until, false)
					So(err, ShouldBeNil)
					So(result, ShouldNotBeNil)

					metricsData := result.GetMetricsData()
					So(len(metricsData), ShouldEqual, 1)
					So(metricsData[0].Name, ShouldEqual, "t1")

					server.Close()
				}
			})
		}
	})
}
