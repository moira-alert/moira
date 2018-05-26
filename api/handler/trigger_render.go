package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/plotting"
)

type metricFormat int

// Supported metrics format types
const (
	PNG metricFormat = iota
	JSON
	RAW
	CARBONAPI
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse from: %s", fromStr)))
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", toStr)))
		return
	}
	format, err := getMetricFormat(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	tts, trigger, err := controller.GetTriggerEvaluationResult(database, int64(from), int64(to), triggerID)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}

	carbonapiRaw := ""

	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
		carbonapiRaw += fmt.Sprintf("IsAbsent: %+v\n", len(ts.IsAbsent))
		carbonapiRaw += fmt.Sprintf("Values: %+v\n", len(ts.Values))
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
		carbonapiRaw += fmt.Sprintf("IsAbsent: %+v\n", len(ts.IsAbsent))
		carbonapiRaw += fmt.Sprintf("Values: %+v\n", len(ts.Values))
	}

	switch format {
	case JSON:
		json := types.MarshalJSON(metricsData)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(json)
	case PNG:
		font, _ := plotting.GetDefaultFont()
		plot := plotting.FromParams(trigger.Name, plotting.DarkTheme, nil, trigger.WarnValue, trigger.ErrorValue)
		renderable := plot.GetRenderable(metricsData, font)
		writer.Header().Set("Content-Type", "image/png")
		renderable.Render(chart.PNG, writer)
	case RAW:
		font, _ := plotting.GetDefaultFont()
		plot := plotting.FromParams(trigger.Name, plotting.DarkTheme, nil, trigger.WarnValue, trigger.ErrorValue)
		renderable := plot.GetRenderable(metricsData, font)
		raw := []byte(fmt.Sprintf("%+v\n", renderable))
		writer.Header().Set("Content-Type", "text")
		writer.Write(raw)
	case CARBONAPI:
		carbonapi := []byte(carbonapiRaw)
		writer.Header().Set("Content-Type", "text")
		writer.Write(carbonapi)
	default:
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("inexpected metrics format")))
	}
}

func getMetricFormat(request *http.Request) (metricFormat, error) {
	format := request.URL.Query().Get("format")
	if format == "" {
		return JSON, nil
	}
	switch format {
	case "json":
		return JSON, nil
	case "png":
		return PNG, nil
	case "raw":
		return RAW, nil
	case "carbonapi":
		return CARBONAPI, nil
	default:
		return JSON, fmt.Errorf("invalid format type: %s", format)
	}
}
