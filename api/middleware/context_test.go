package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const expectedBadRequest = `{"status":"Invalid request","error":"invalid URL escape \"%\""}
`

func TestPaginateMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultPage := int64(1)
		defaultSize := int64(10)

		Convey("with correct parameters", func() {
			parameters := []string{"p=0&size=100", "p=0", "size=100", "", "p=test&size=100", "p=0&size=test"}

			for _, param := range parameters {
				testRequest := httptest.NewRequest(http.MethodGet, "/test?"+param, nil)
				handler := func(w http.ResponseWriter, r *http.Request) {}

				middlewareFunc := Paginate(defaultPage, defaultSize)
				wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

				wrappedHandler.ServeHTTP(responseWriter, testRequest)
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?p=0%&size=100", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := Paginate(defaultPage, defaultSize)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestPagerMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultCreatePager := false
		defaultPagerID := "test"

		Convey("with correct parameters", func() {
			parameters := []string{"pagerID=test&createPager=true", "pagerID=test", "createPager=true", "", "pagerID=-1&createPager=true", "pagerID=test&createPager=-1"}

			for _, param := range parameters {
				testRequest := httptest.NewRequest(http.MethodGet, "/test?"+param, nil)
				handler := func(w http.ResponseWriter, r *http.Request) {}

				middlewareFunc := Pager(defaultCreatePager, defaultPagerID)
				wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

				wrappedHandler.ServeHTTP(responseWriter, testRequest)
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?pagerID=test%&createPager=true", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := Pager(defaultCreatePager, defaultPagerID)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestPopulateMiddleware(t *testing.T) {
	Convey("checking correctness of parameter", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultPopulated := false

		Convey("with correct parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?populated=true", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := Populate(defaultPopulated)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?populated%=true", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := Populate(defaultPopulated)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestDateRangeMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultFrom := "-1hour"
		defaultTo := "now"

		Convey("with correct parameters", func() {
			parameters := []string{"from=-2hours&to=now", "from=-2hours", "to=now", "", "from=-2&to=now", "from=-2hours&to=-1"}

			for _, param := range parameters {
				testRequest := httptest.NewRequest(http.MethodGet, "/test?"+param, nil)
				handler := func(w http.ResponseWriter, r *http.Request) {}

				middlewareFunc := DateRange(defaultFrom, defaultTo)
				wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

				wrappedHandler.ServeHTTP(responseWriter, testRequest)
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?from=-2hours%&to=now", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := DateRange(defaultFrom, defaultTo)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestTargetNameMiddleware(t *testing.T) {
	Convey("checking correctness of parameter", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultTargetName := "test"

		Convey("with correct parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?target=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := TargetName(defaultTargetName)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?target%=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := TargetName(defaultTargetName)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
