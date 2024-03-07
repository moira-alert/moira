package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/moira-alert/go-chart"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/plotting"
)

// nolint: gofmt,goimports.
//
//	@summary	Render trigger metrics plot.
//	@id			render-trigger-metrics.
//	@tags		trigger.
//	@produce	png.
//	@param		triggerID	path	string	true	"Trigger ID"						default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c).
//	@param		target		query	string	false	"Target metric name"				default(t1).
//	@param		from		query	string	false	"Start time for metrics retrieval"	default(-1hour).
//	@param		to			query	string	false	"End time for metrics retrieval"	default(now).
//	@param		timezone	query	string	false	"Timezone for rendering"			default(UTC).
//	@param		theme		query	string	false	"Plot theme"						default(light).
//	@param		realtime	query	bool	false	"Fetch real-time data"				default(false).
//	@success	200			"Rendered plot image successfully".
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client".
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found".
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error".
//	@router		/trigger/{triggerID}/render [get].
func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	sourceProvider, targetName, from, to, triggerID, fetchRealtimeData, err := getEvaluationParameters(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	metricsData, trigger, err := evaluateTargetMetrics(sourceProvider, from, to, triggerID, fetchRealtimeData)
	if err != nil {
		if trigger == nil {
			render.Render(writer, request, api.ErrorNotFound(fmt.Sprintf("trigger with ID = '%s' does not exists", triggerID))) //nolint
		} else {
			render.Render(writer, request, api.ErrorInternalServer(err)) //nolint
		}
		return
	}

	targetMetrics, ok := metricsData[targetName]
	if !ok {
		render.Render(writer, request, api.ErrorNotFound(fmt.Sprintf("Cannot find target %s", targetName))) //nolint
	}

	renderable, err := buildRenderable(request, trigger, targetMetrics, targetName)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err)) //nolint
		return
	}
	writer.Header().Set("Content-Type", "image/png")
	err = renderable.Render(chart.PNG, writer)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("can not render plot %s", err.Error()))) //nolint
	}
}

func getEvaluationParameters(request *http.Request) (sourceProvider *metricSource.SourceProvider, targetName string, from int64, to int64, triggerID string, fetchRealtimeData bool, err error) {
	sourceProvider = middleware.GetTriggerTargetsSourceProvider(request)
	targetName = middleware.GetTargetName(request)
	triggerID = middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from = date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	urlValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("failed to parse query string: %w", err)
	}

	if from == 0 {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("can not parse from: %s", fromStr)
	}
	from -= from % 60 //nolint

	to = date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("can not parse to: %s", fromStr)
	}

	realtime := urlValues.Get("realtime")
	if realtime == "" {
		return
	}

	fetchRealtimeData, err = strconv.ParseBool(realtime)
	if err != nil {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("invalid realtime param: %s", err.Error())
	}

	return
}

func evaluateTargetMetrics(metricSourceProvider *metricSource.SourceProvider, from, to int64, triggerID string, fetchRealtimeData bool) (map[string][]metricSource.MetricData, *moira.Trigger, error) {
	tts, trigger, err := controller.GetTriggerEvaluationResult(database, metricSourceProvider, from, to, triggerID, fetchRealtimeData)
	if err != nil {
		return nil, trigger, err
	}

	return tts, trigger, err
}

func buildRenderable(request *http.Request, trigger *moira.Trigger, metricsData []metricSource.MetricData, targetName string) (*chart.Chart, error) {
	urlValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query string: %w", err)
	}

	timezone := urlValues.Get("timezone")
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s timezone: %s", timezone, err.Error())
	}

	plotTheme := urlValues.Get("theme")
	plotTemplate, err := plotting.GetPlotTemplate(plotTheme, location)
	if err != nil {
		return nil, fmt.Errorf("can not initialize plot theme %s", err.Error())
	}

	renderable, err := plotTemplate.GetRenderable(targetName, trigger, metricsData)
	if err != nil {
		return nil, err
	}

	return &renderable, err
}
