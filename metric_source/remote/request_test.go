package remote

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mock_clock "github.com/moira-alert/moira/mock/clock"

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
		actual, isRemoteAvailable, err := remote.makeRequest(request)
		So(err, ShouldBeNil)
		So(isRemoteAvailable, ShouldBeTrue)
		So(actual, ShouldResemble, body)
	})

	Convey("Client returns status InternalServerError", t, func() {
		server := createServer(body, http.StatusInternalServerError)
		remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
		request, _ := remote.prepareRequest(from, until, target)
		actual, isRemoteAvailable, err := remote.makeRequest(request)
		So(err, ShouldResemble, fmt.Errorf("remote server response status %d: %s", http.StatusInternalServerError, string(body)))
		So(isRemoteAvailable, ShouldBeTrue)
		So(actual, ShouldResemble, body)
	})

	Convey("Client calls bad url", t, func() {
		server := createServer(body, http.StatusOK)
		client := server.Client()
		client.Timeout = time.Millisecond
		remote := Remote{client: client, config: &Config{URL: "http://bad/"}}
		request, _ := remote.prepareRequest(from, until, target)
		actual, isRemoteAvailable, err := remote.makeRequest(request)
		So(err, ShouldNotBeEmpty)
		So(isRemoteAvailable, ShouldBeFalse)
		So(actual, ShouldBeEmpty)
	})

	Convey("Client returns status Remote Unavailable status codes", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			server := createServer(body, statusCode)
			remote := Remote{client: server.Client(), config: &Config{URL: server.URL}}
			request, _ := remote.prepareRequest(from, until, target)
			actual, isRemoteAvailable, err := remote.makeRequest(request)
			So(err, ShouldResemble, fmt.Errorf(
				"the remote server is not available. Response status %d: %s", statusCode, string(body),
			))
			So(isRemoteAvailable, ShouldBeFalse)
			So(actual, ShouldResemble, body)
		}
	})
}

func TestMakeRequestWithRetries(t *testing.T) {
	var from int64 = 300
	var until int64 = 500
	target := "foo.bar"
	body := []byte("Some string")
	testConfigs := []Config{
		{},
		{RetrySeconds: []time.Duration{time.Second}},
		{RetrySeconds: []time.Duration{time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second}},
		{RetrySeconds: []time.Duration{time.Second, time.Second, time.Second}},
	}
	mockCtrl := gomock.NewController(t)

	Convey("Given server returns OK response", t, func() {
		server := createTestServer(TestResponse{body, http.StatusOK})
		systemClock := mock_clock.NewMockClock(mockCtrl)
		systemClock.EXPECT().Sleep(time.Second).Times(0)

		Convey("request is successful", func() {
			remote := Remote{client: server.Client(), clock: systemClock}

			for _, config := range testConfigs {
				config.URL = server.URL
				remote.config = &config
				request, _ := remote.prepareRequest(from, until, target)
				actual, isRemoteAvailable, err := remote.makeRequestWithRetries(
					request,
					remote.config.Timeout,
					remote.config.RetrySeconds,
				)
				So(err, ShouldBeNil)
				So(isRemoteAvailable, ShouldBeTrue)
				So(actual, ShouldResemble, body)
			}
		})
	})

	Convey("Given server returns 500 response", t, func() {
		server := createTestServer(TestResponse{body, http.StatusInternalServerError})
		systemClock := mock_clock.NewMockClock(mockCtrl)
		systemClock.EXPECT().Sleep(time.Second).Times(0)

		Convey("request failed with 500 response and remote is available", func() {
			remote := Remote{client: server.Client(), clock: systemClock}

			for _, config := range testConfigs {
				config.URL = server.URL
				remote.config = &config
				request, _ := remote.prepareRequest(from, until, target)
				actual, isRemoteAvailable, err := remote.makeRequestWithRetries(
					request,
					remote.config.Timeout,
					remote.config.RetrySeconds,
				)
				So(err, ShouldResemble, fmt.Errorf("remote server response status %d: %s", http.StatusInternalServerError, string(body)))
				So(isRemoteAvailable, ShouldBeTrue)
				So(actual, ShouldResemble, body)
			}
		})
	})

	Convey("Given client calls bad url", t, func() {
		server := createTestServer(TestResponse{body, http.StatusOK})

		Convey("request failed and remote is unavailable", func() {
			remote := Remote{client: server.Client()}
			for _, config := range testConfigs {
				config.URL = "http://bad/"
				remote.config = &config
				systemClock := mock_clock.NewMockClock(mockCtrl)
				systemClock.EXPECT().Sleep(time.Second).Times(len(config.RetrySeconds))
				remote.clock = systemClock

				request, _ := remote.prepareRequest(from, until, target)
				actual, isRemoteAvailable, err := remote.makeRequestWithRetries(
					request,
					time.Millisecond,
					remote.config.RetrySeconds,
				)
				So(err, ShouldNotBeEmpty)
				So(isRemoteAvailable, ShouldBeFalse)
				So(actual, ShouldBeEmpty)
			}
		})
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
					systemClock.EXPECT().Sleep(time.Second).Times(len(config.RetrySeconds))
					remote.clock = systemClock

					request, _ := remote.prepareRequest(from, until, target)
					actual, isRemoteAvailable, err := remote.makeRequestWithRetries(
						request,
						remote.config.Timeout,
						remote.config.RetrySeconds,
					)
					So(err, ShouldResemble, fmt.Errorf(
						"the remote server is not available. Response status %d: %s", statusCode, string(body),
					))
					So(isRemoteAvailable, ShouldBeFalse)
					So(actual, ShouldBeNil)
				}
			})
		}
	})

	Convey("Given server returns Remote Unavailable response temporary", t, func() {
		for _, statusCode := range remoteUnavailableStatusCodes {
			Convey(fmt.Sprintf(
				"request is successful with retry after %d response and remote is available", statusCode,
			), func() {
				for _, config := range testConfigs {
					if len(config.RetrySeconds) == 0 {
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

					request, _ := remote.prepareRequest(from, until, target)
					actual, isRemoteAvailable, err := remote.makeRequestWithRetries(
						request,
						remote.config.Timeout,
						remote.config.RetrySeconds,
					)
					So(err, ShouldBeNil)
					So(isRemoteAvailable, ShouldBeTrue)
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

var remoteUnavailableStatusCodes = []int{
	http.StatusUnauthorized,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}
