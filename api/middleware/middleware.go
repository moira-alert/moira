package middleware

import (
	"context"
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type contextKey string

func (key contextKey) String() string {
	return "api context key " + string(key)
}

var (
	databaseKey        contextKey = "database"
	triggerIDKey       contextKey = "triggerID"
	tagKey             contextKey = "tag"
	subscriptionIDKey  contextKey = "subscriptionID"
	pageKey            contextKey = "page"
	sizeKey            contextKey = "size"
	fromKey            contextKey = "from"
	toKey              contextKey = "to"
	loginKey           contextKey = "login"
	timeSeriesNamesKey contextKey = "timeSeriesNames"
)

func GetDatabase(request *http.Request) moira.Database {
	return request.Context().Value(databaseKey).(moira.Database)
}

func GetLogin(request *http.Request) string {
	return request.Context().Value(loginKey).(string)
}

func GetTriggerID(request *http.Request) string {
	return request.Context().Value(triggerIDKey).(string)
}

func GetTag(request *http.Request) string {
	return request.Context().Value(tagKey).(string)
}

func GetSubscriptionID(request *http.Request) string {
	return request.Context().Value(subscriptionIDKey).(string)
}

func GetPage(request *http.Request) int64 {
	return request.Context().Value(pageKey).(int64)
}

func GetSize(request *http.Request) int64 {
	return request.Context().Value(sizeKey).(int64)
}

func GetFromStr(request *http.Request) string {
	return request.Context().Value(fromKey).(string)
}

func GetToStr(request *http.Request) string {
	return request.Context().Value(toKey).(string)
}

func SetTimeSeriesNames(request *http.Request, timeSeriesNames map[string]bool) {
	ctx := context.WithValue(request.Context(), timeSeriesNamesKey, timeSeriesNames)
	*request = *request.WithContext(ctx)
}

func GetTimeSeriesNames(request *http.Request) map[string]bool {
	return request.Context().Value(timeSeriesNamesKey).(map[string]bool)
}
