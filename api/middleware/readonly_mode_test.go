package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moira-alert/moira/api"
	. "github.com/smartystreets/goconvey/convey"
)

func TestReadonlyModeMiddleware(t *testing.T) {
	Convey("Given readonly mode is disabled", t, func() {
		config := &api.Config{Flags: api.FeatureFlags{IsReadonlyEnabled: false}}

		Convey("Performing get request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodGet)

			So(actual, ShouldEqual, http.StatusOK)
		})
		Convey("Performing put request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodPut)

			So(actual, ShouldEqual, http.StatusOK)
		})
	})

	Convey("Given readonly mode is enabled", t, func() {
		config := &api.Config{Flags: api.FeatureFlags{IsReadonlyEnabled: true}}

		Convey("Performing get request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodGet)

			So(actual, ShouldEqual, http.StatusOK)
		})
		Convey("Performing put request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodPut)

			So(actual, ShouldEqual, http.StatusForbidden)
		})
		Convey("Performing post request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodPost)

			So(actual, ShouldEqual, http.StatusForbidden)
		})
		Convey("Performing patch request", func() {
			actual := PerformRequestWithReadonlyModeMiddleware(config, http.MethodPatch)

			So(actual, ShouldEqual, http.StatusForbidden)
		})
	})
}

func PerformRequestWithReadonlyModeMiddleware(config *api.Config, method string) int {
	responseWriter := httptest.NewRecorder()

	testRequest := httptest.NewRequest(method, "/test", nil)

	handler := func(w http.ResponseWriter, r *http.Request) {}
	middlewareFunc := ReadOnlyMiddleware(config)
	wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

	wrappedHandler.ServeHTTP(responseWriter, testRequest)
	response := responseWriter.Result()
	defer response.Body.Close()

	return response.StatusCode
}
