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

func testRequestOk(
	url string,
	middlewareFunc func(next http.Handler) http.Handler,
) {
	responseWriter := httptest.NewRecorder()

	testRequest := httptest.NewRequest(http.MethodGet, url, nil)
	handler := func(w http.ResponseWriter, r *http.Request) {}

	wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

	wrappedHandler.ServeHTTP(responseWriter, testRequest)
	response := responseWriter.Result()
	defer response.Body.Close()

	So(response.StatusCode, ShouldEqual, http.StatusOK)
}

func testRequestFails(
	url string,
	middlewareFunc func(next http.Handler) http.Handler,
	failedRequestStr string,
	failedRequestStatusCode int,
) {
	responseWriter := httptest.NewRecorder()

	testRequest := httptest.NewRequest(http.MethodGet, url, nil)
	handler := func(w http.ResponseWriter, r *http.Request) {}

	wrappedHandler := middlewareFunc(http.HandlerFunc(handler))

	wrappedHandler.ServeHTTP(responseWriter, testRequest)
	response := responseWriter.Result()
	defer response.Body.Close()
	contentBytes, _ := io.ReadAll(response.Body)
	contents := string(contentBytes)

	So(contents, ShouldEqual, failedRequestStr)
	So(response.StatusCode, ShouldEqual, failedRequestStatusCode)
}

func TestPaginateMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		//responseWriter := httptest.NewRecorder()
		defaultPage := int64(1)
		defaultSize := int64(10)

		Convey("with correct parameters", func() {
			parameters := []string{"p=0&size=100", "p=0", "size=100", "", "p=test&size=100", "p=0&size=test"}

			for _, param := range parameters {
				testRequestOk(
					"/test?"+param,
					Paginate(defaultPage, defaultSize))
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequestFails(
				"/test?p=0%&size=100",
				Paginate(defaultPage, defaultSize),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestPagerMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		defaultCreatePager := false
		defaultPagerID := "test"

		Convey("with correct parameters", func() {
			parameters := []string{"pagerID=test&createPager=true", "pagerID=test", "createPager=true", "", "pagerID=-1&createPager=true", "pagerID=test&createPager=-1"}

			for _, param := range parameters {
				testRequestOk(
					"/test?"+param,
					Pager(defaultCreatePager, defaultPagerID))
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequestFails(
				"/test?pagerID=test%&createPager=true",
				Pager(defaultCreatePager, defaultPagerID),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestPopulateMiddleware(t *testing.T) {
	Convey("checking correctness of parameter", t, func() {
		defaultPopulated := false

		Convey("with correct parameter", func() {
			testRequestOk(
				"/test?populated=true",
				Populate(defaultPopulated))
		})

		Convey("with wrong url query parameter", func() {
			testRequestFails(
				"/test?populated%=true",
				Populate(defaultPopulated),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestDateRangeMiddleware(t *testing.T) {
	Convey("checking correctness of parameters", t, func() {
		defaultFrom := "-1hour"
		defaultTo := "now"

		Convey("with correct parameters", func() {
			parameters := []string{"from=-2hours&to=now", "from=-2hours", "to=now", "", "from=-2&to=now", "from=-2hours&to=-1"}

			for _, param := range parameters {
				testRequestOk(
					"/test?"+param,
					DateRange(defaultFrom, defaultTo))
			}
		})

		Convey("with wrong url query parameters", func() {
			testRequestFails(
				"/test?from=-2hours%&to=now",
				DateRange(defaultFrom, defaultTo),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestTargetNameMiddleware(t *testing.T) {
	Convey("checking correctness of parameter", t, func() {
		defaultTargetName := "test"

		Convey("with correct parameter", func() {
			testRequestOk(
				"/test?target=test",
				TargetName(defaultTargetName))
		})

		Convey("with wrong url query parameter", func() {
			testRequestFails(
				"/test?target%=test",
				TargetName(defaultTargetName),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestMetricContextMiddleware(t *testing.T) {
	Convey("Check metric provider", t, func() {
		defaultMetric := ".*"

		Convey("status ok with correct query paramete", func() {
			testRequestOk(
				"/test?metric=test%5C.metric.*",
				MetricContext(defaultMetric))
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequestFails(
				"/test?metric%=test",
				MetricContext(defaultMetric),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestStatesContextMiddleware(t *testing.T) {
	Convey("Checking states provide", t, func() {
		Convey("ok with correct states list", func() {
			testRequestOk(
				"/test?states=OK%2CERROR",
				StatesContext())
		})

		Convey("bad request with bad states list", func() {
			testRequestFails(
				"/test?states=OK%2CERROR%2Cwarn",
				StatesContext(),
				"{\"status\":\"Invalid request\",\"error\":\"bad state in query parameter: warn\"}\n",
				http.StatusBadRequest)
		})

		Convey("bad request with wrong url query parameter", func() {
			testRequestFails(
				"/test?states%=test",
				StatesContext(),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}

func TestSearchTextContext(t *testing.T) {
	Convey("Checkins search text context", t, func() {
		defaultSearchText := regexp.MustCompile(".*")

		Convey("status ok with correct query parameter", func() {
			testRequestOk(
				"/test?searchText=test%5Ctext.*",
				SearchTextContext(defaultSearchText))
		})

		Convey("status ok with empty query parameter", func() {
			testRequestOk(
				"/test?searchText=",
				SearchTextContext(defaultSearchText))
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequestFails(
				"/test?searchText%=test",
				SearchTextContext(defaultSearchText),
				expectedBadRequest,
				http.StatusBadRequest)
		})

		Convey("status bad request with bad regexp", func() {
			testRequestFails(
				"/test?searchText=*",
				SearchTextContext(defaultSearchText),
				"{\"status\":\"Invalid request\",\"error\":\"failed to parse searchText template '*': error parsing regexp: missing argument to repetition operator: `*`\"}\n",
				http.StatusBadRequest)
		})
	})
}

func TestSortOrderContext(t *testing.T) {
	Convey("Checking sort order context", t, func() {
		defaultSortOrder := api.NoSortOrder

		Convey("with correct query parameter", func() {
			sortOrders := []api.SortOrder{api.NoSortOrder, api.AscSortOrder, api.DescSortOrder, "some"}

			for i, givenSortOrder := range sortOrders {
				Convey(fmt.Sprintf("case %d: sord order '%s'", i+1, givenSortOrder), func() {
					testRequestOk(
						fmt.Sprintf("/test?sort=%s", givenSortOrder),
						SortOrderContext(defaultSortOrder))
				})
			}
		})

		Convey("status bad request with wrong url query parameter", func() {
			testRequestFails(
				"/test?sort%=test",
				SortOrderContext(defaultSortOrder),
				expectedBadRequest,
				http.StatusBadRequest)
		})
	})
}
