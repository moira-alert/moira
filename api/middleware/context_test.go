package middleware

import (
	"fmt"
	"github.com/moira-alert/moira/api"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestMetricContextMiddleware(t *testing.T) {
	Convey("Check metric provider", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultMetric := ".*"

		Convey("status ok with correct query paramete", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?metric=test%5C.metric.*", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := MetricContext(defaultMetric)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?metric%=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := MetricContext(defaultMetric)
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

func TestStatesContextMiddleware(t *testing.T) {
	Convey("Checking states provide", t, func() {
		responseWriter := httptest.NewRecorder()

		Convey("ok with correct states list", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?states=OK%2CERROR", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := StatesContext()
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("bad request with bad states list", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?states=OK%2CERROR%2Cwarn", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := StatesContext()
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("bad request with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?states%=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := StatesContext()
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

func TestSearchTextContext(t *testing.T) {
	Convey("Checkins search text context", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultSearchText := regexp.MustCompile(".*")

		Convey("status ok with correct query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?searchText=test%5Ctext.*", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := SearchTextContext(defaultSearchText)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("status ok with empty query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?searchText=", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := SearchTextContext(defaultSearchText)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?searchText%=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := SearchTextContext(defaultSearchText)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, expectedBadRequest)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("status bad request with bad regexp", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?searchText=*", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := SearchTextContext(defaultSearchText)
			wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

			wrappedHandler.ServeHTTP(responseWriter, testRequest)
			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, "{\"status\":\"Invalid request\",\"error\":\"failed to parse searchText template '*': error parsing regexp: missing argument to repetition operator: `*`\"}\n")
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestSortOrderContext(t *testing.T) {
	Convey("Checking sort order context", t, func() {
		responseWriter := httptest.NewRecorder()
		defaultSortOrder := api.NoSortOrder

		Convey("with correct query parameter", func() {
			sortOrders := []api.SortOrder{api.NoSortOrder, api.AscSortOrder, api.DescSortOrder, "some"}

			for i, givenSortOrder := range sortOrders {
				Convey(fmt.Sprintf("case %d: sord order '%s'", i+1, givenSortOrder), func() {
					testRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test?sort=%s", givenSortOrder), nil)
					handler := func(w http.ResponseWriter, r *http.Request) {}

					middlewareFunc := SortOrderContext(defaultSortOrder)
					wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

					wrappedHandler.ServeHTTP(responseWriter, testRequest)
					response := responseWriter.Result()
					defer response.Body.Close()

					So(response.StatusCode, ShouldEqual, http.StatusOK)
				})
			}
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/test?sort%=test", nil)
			handler := func(w http.ResponseWriter, r *http.Request) {}

			middlewareFunc := SortOrderContext(defaultSortOrder)
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
