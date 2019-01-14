package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
)

func triggers(metricSourceProvider *metricSource.SourceProvider, searcher moira.Searcher) func(chi.Router) {
	return func(router chi.Router) {
		router.Use(middleware.MetricSourceProvider(metricSourceProvider))
		router.Use(middleware.SearchIndexContext(searcher))
		router.Get("/", getAllTriggers)
		router.Put("/", createTrigger)
		router.Route("/{triggerId}", trigger)
		router.With(middleware.Paginate(0, 10)).Get("/search", searchTriggers)
		// ToDo: DEPRECATED method. Remove in Moira 2.6
		router.With(middleware.Paginate(0, 10)).Get("/page", searchTriggers)
	}
}

func getAllTriggers(writer http.ResponseWriter, request *http.Request) {
	triggersList, errorResponse := controller.GetAllTriggers(database)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func createTrigger(writer http.ResponseWriter, request *http.Request) {
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) {
		case local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid graphite targets: %s", err.Error())))
		case expression.ErrInvalidExpression:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid expression: %s", err.Error())))
		case remote.ErrRemoteTriggerResponse:
			render.Render(writer, request, api.ErrorRemoteServerUnavailable(err))
		default:
			render.Render(writer, request, api.ErrorInternalServer(err))
		}
		return
	}
	timeSeriesNames := middleware.GetTimeSeriesNames(request)
	response, err := controller.CreateTrigger(database, &trigger.TriggerModel, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func searchTriggers(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	onlyErrors := getOnlyProblemsFlag(request)
	filterTags := getRequestTags(request)
	searchString := getSearchRequestString(request)

	page := middleware.GetPage(request)
	size := middleware.GetSize(request)

	triggersList, errorResponse := controller.SearchTriggers(database, searchIndex, page, size, onlyErrors, filterTags, searchString)
	if errorResponse != nil {
		render.Render(writer, request, errorResponse)
		return
	}

	if err := render.Render(writer, request, triggersList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getRequestTags(request *http.Request) []string {
	var filterTags []string
	i := 0
	for {
		tag := request.FormValue(fmt.Sprintf("tags[%v]", i))
		if tag == "" {
			break
		}
		filterTags = append(filterTags, tag)
		i++
	}
	return filterTags
}

func getOnlyProblemsFlag(request *http.Request) bool {
	onlyProblemsStr := request.FormValue("onlyProblems")
	if onlyProblemsStr != "" {
		onlyProblems, _ := strconv.ParseBool(onlyProblemsStr)
		return onlyProblems
	}
	return false
}

func getSearchRequestString(request *http.Request) string {
	searchText := request.FormValue("text")
	searchText, _ = url.PathUnescape(searchText)
	return searchText
}
