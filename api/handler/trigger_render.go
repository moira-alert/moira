package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

type metricFormat int

// Supported metrics format types
const (
	PNG metricFormat = iota
	PNGr
	JSON
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

	var metricsData = make([]*expr.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}

	switch format {
	case JSON:
		json := expr.MarshalJSON(metricsData)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(json)
	case PNG:
		params := getPictureParams()
		params.Title = trigger.Name
		png := expr.MarshalPNG(params, metricsData)
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(png)
	case PNGr:
		png := expr.MarshalPNGRequest(request, metricsData)
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(png)
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
	case "pngr":
		return PNGr, nil
	default:
		return JSON, fmt.Errorf("invalid format type: %s", format)
	}
}

func getPictureParams() expr.PictureParams {
	params := expr.DefaultParams
	params.Width = 586
	params.Height = 380
	params.LeftWidth = 2
	params.BgColor = "1f1d1d"
	params.MinorGridLineColor = "1f1d1d"
	params.MajorGridLineColor = "grey"
	return params
}
