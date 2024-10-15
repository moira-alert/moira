package remote

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moira-alert/moira/metric_source/retries"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
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

func Test_requestToRemoteGraphite_DoRetryableOperation(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	body := []byte("Some string")

	testTimeout := time.Millisecond * 10

	Convey("Client returns status OK", t, func() {
		server := createServer(body, http.StatusOK)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		request, _ := remote.prepareRequest(from, until, target)

		retryableOp := requestToRemoteGraphite{
			request:        request,
			client:         remote.client,
			requestTimeout: testTimeout,
		}

		actual, err := retryableOp.DoRetryableOperation()

		So(err, ShouldBeNil)
		So(actual, ShouldResemble, body)
	})

	Convey("Client returns status InternalServerError", t, func() {
		server := createServer(body, http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		request, _ := remote.prepareRequest(from, until, target)

		retryableOp := requestToRemoteGraphite{
			request:        request,
			client:         remote.client,
			requestTimeout: testTimeout,
		}

		actual, err := retryableOp.DoRetryableOperation()

		So(err, ShouldResemble, backoff.Permanent(errInvalidRequest{
			internalErr: fmt.Errorf("bad response status %d: %s", http.StatusInternalServerError, string(body)),
		}))
		So(actual, ShouldResemble, body)
	})

	Convey("Client calls bad url", t, func() {
		server := createServer(body, http.StatusOK)
		client := server.Client()
		remote := Remote{client: client, config: &Config{URL: "http://bad/"}}
		request, _ := remote.prepareRequest(from, until, target)

		retryableOp := requestToRemoteGraphite{
			request:        request,
			client:         remote.client,
			requestTimeout: testTimeout,
		}

		actual, err := retryableOp.DoRetryableOperation()

		So(err, ShouldHaveSameTypeAs, errRemoteUnavailable{})
		So(actual, ShouldBeEmpty)
	})

	Convey("Client returns status Remote Unavailable status codes", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			server := createServer(body, statusCode)
			remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
			request, _ := remote.prepareRequest(from, until, target)

			retryableOp := requestToRemoteGraphite{
				request:        request,
				client:         remote.client,
				requestTimeout: testTimeout,
			}

			actual, err := retryableOp.DoRetryableOperation()

			So(err, ShouldResemble, errRemoteUnavailable{
				internalErr: fmt.Errorf(
					"the remote server is not available. Response status %d: %s", statusCode, string(body)),
			})
			So(actual, ShouldResemble, body)
		}
	})
}

func TestMakeRequestWithRetries(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	body := []byte("Some string")
	testConfigs := []*Config{
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
	}
	mockCtrl := gomock.NewController(t)
	retrier := retries.NewStandardRetrier[[]byte]()

	Convey("Given server returns OK response", t, func() {
		server := createTestServer(TestResponse{body, http.StatusOK})
		systemClock := mock_clock.NewMockClock(mockCtrl)
		systemClock.EXPECT().Sleep(time.Second).Times(0)

		Convey("request is successful", func() {
			remote := Remote{
				client:  server.Client(),
				retrier: retrier,
			}

			for _, config := range testConfigs {
				config.URL = server.URL
				remote.config = config
				request, _ := remote.prepareRequest(from, until, target)
				backoffPolicy := retries.NewExponentialBackoffFactory(config.Retries).NewBackOff()

				actual, err := remote.makeRequest(
					request,
					remote.config.Timeout,
					backoffPolicy,
				)

				So(err, ShouldBeNil)
				So(actual, ShouldResemble, body)
			}
		})
	})

	Convey("Given server returns 500 response", t, func() {
		server := createTestServer(TestResponse{body, http.StatusInternalServerError})
		expectedErr := errInvalidRequest{
			internalErr: fmt.Errorf("bad response status %d: %s", http.StatusInternalServerError, string(body)),
		}
		// systemClock.EXPECT().Sleep(time.Second).Times(0)

		Convey("request failed with 500 response and remote is available", func() {
			remote := Remote{
				client:  server.Client(),
				retrier: retrier,
			}

			for _, config := range testConfigs {
				config.URL = server.URL
				remote.config = config
				request, _ := remote.prepareRequest(from, until, target)
				backoffPolicy := retries.NewExponentialBackoffFactory(config.Retries).NewBackOff()

				actual, err := remote.makeRequest(
					request,
					remote.config.Timeout,
					backoffPolicy,
				)

				So(err, ShouldResemble, expectedErr)
				So(actual, ShouldResemble, body)
			}
		})
	})

	Convey("Given client calls bad url", t, func() {
		server := createTestServer(TestResponse{body, http.StatusOK})

		Convey("request failed and remote is unavailable", func() {
			remote := Remote{
				client:  server.Client(),
				retrier: retrier,
			}

			for _, config := range testConfigs {
				config.URL = "http://bad/"
				remote.config = config

				request, _ := remote.prepareRequest(from, until, target)
				backoffPolicy := retries.NewExponentialBackoffFactory(config.Retries).NewBackOff()

				actual, err := remote.makeRequest(
					request,
					remote.config.Timeout,
					backoffPolicy,
				)

				So(err, ShouldHaveSameTypeAs, errRemoteUnavailable{})
				So(actual, ShouldBeEmpty)
			}
		})
	})

	Convey("Given server returns Remote Unavailable responses permanently", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			server := createTestServer(TestResponse{body, statusCode})

			Convey(fmt.Sprintf(
				"request failed with %d response status code and remote is unavailable", statusCode,
			), func() {
				remote := Remote{
					client:  server.Client(),
					retrier: retrier,
				}

				for _, config := range testConfigs {
					config.URL = server.URL
					remote.config = config

					request, _ := remote.prepareRequest(from, until, target)
					backoffPolicy := retries.NewExponentialBackoffFactory(config.Retries).NewBackOff()

					actual, err := remote.makeRequest(
						request,
						remote.config.Timeout,
						backoffPolicy,
					)

					So(err, ShouldResemble, errRemoteUnavailable{
						internalErr: fmt.Errorf(
							"the remote server is not available. Response status %d: %s", statusCode, string(body),
						),
					})
					So(actual, ShouldResemble, body)
				}
			})
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"request is successful with retry after %d response and remote is available", statusCode,
			), func() {
				for _, config := range testConfigs {
					server := createTestServer(
						TestResponse{body, statusCode},
						TestResponse{body, http.StatusOK},
					)
					config.URL = server.URL
					remote := Remote{
						client:  server.Client(),
						config:  config,
						retrier: retrier,
					}

					request, _ := remote.prepareRequest(from, until, target)
					backoffPolicy := retries.NewExponentialBackoffFactory(config.Retries).NewBackOff()

					actual, err := remote.makeRequest(
						request,
						remote.config.Timeout,
						backoffPolicy,
					)

					So(err, ShouldBeNil)
					So(actual, ShouldResemble, body)
				}
			})
		}
	})
}

func createServer(body []byte, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(statusCode)
		rw.Write(body) //nolint
	}))
}

func createTestServer(testResponses ...TestResponse) *httptest.Server {
	responseWriter := NewTestResponseWriter(testResponses)
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		response := responseWriter.GetResponse()
		rw.WriteHeader(response.statusCode)
		rw.Write(response.body) //nolint
	}))
}

type TestResponse struct {
	body       []byte
	statusCode int
}

type TestResponseWriter struct {
	responses []TestResponse
	count     int
}

func NewTestResponseWriter(testResponses []TestResponse) *TestResponseWriter {
	responseWriter := new(TestResponseWriter)
	responseWriter.responses = testResponses
	responseWriter.count = 0
	return responseWriter
}

func (responseWriter *TestResponseWriter) GetResponse() TestResponse {
	response := responseWriter.responses[responseWriter.count%len(responseWriter.responses)]
	responseWriter.count++
	return response
}
