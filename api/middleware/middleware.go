package middleware

import (
	"context"
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/remote"
)

// ContextKey used as key of api request context values
type ContextKey string

func (key ContextKey) String() string {
	return "api context key " + string(key)
}

var (
	databaseKey        ContextKey = "database"
	triggerIDKey       ContextKey = "triggerID"
	contactIDKey       ContextKey = "contactID"
	tagKey             ContextKey = "tag"
	subscriptionIDKey  ContextKey = "subscriptionID"
	pageKey            ContextKey = "page"
	sizeKey            ContextKey = "size"
	fromKey            ContextKey = "from"
	toKey              ContextKey = "to"
	loginKey           ContextKey = "login"
	timeSeriesNamesKey ContextKey = "timeSeriesNames"
	remoteConfigKey    ContextKey = "remoteConfig"
)

// GetDatabase gets moira.Database realization from request context
func GetDatabase(request *http.Request) moira.Database {
	return request.Context().Value(databaseKey).(moira.Database)
}

// GetLogin gets user login string from request context, which was sets in UserContext middleware
func GetLogin(request *http.Request) string {
	return request.Context().Value(loginKey).(string)
}

// GetTriggerID gets TriggerID string from request context, which was sets in TriggerContext middleware
func GetTriggerID(request *http.Request) string {
	return request.Context().Value(triggerIDKey).(string)
}

// GetTag gets tag string from request context, which was sets in TagContext middleware
func GetTag(request *http.Request) string {
	return request.Context().Value(tagKey).(string)
}

// GetSubscriptionID gets subscriptionId string from request context, which was sets in SubscriptionContext middleware
func GetSubscriptionID(request *http.Request) string {
	return request.Context().Value(subscriptionIDKey).(string)
}

// GetContactID gets ContactID string from request context, which was sets in TriggerContext middleware
func GetContactID(request *http.Request) string {
	return request.Context().Value(contactIDKey).(string)
}

// GetPage gets page value from request context, which was sets in Paginate middleware
func GetPage(request *http.Request) int64 {
	return request.Context().Value(pageKey).(int64)
}

// GetSize gets size value from request context, which was sets in Paginate middleware
func GetSize(request *http.Request) int64 {
	return request.Context().Value(sizeKey).(int64)
}

// GetFromStr gets 'from' value from request context, which was sets in DateRange middleware
func GetFromStr(request *http.Request) string {
	return request.Context().Value(fromKey).(string)
}

// GetToStr gets 'to' value from request context, which was sets in DateRange middleware
func GetToStr(request *http.Request) string {
	return request.Context().Value(toKey).(string)
}

// SetTimeSeriesNames sets to requests context timeSeriesNames from saved trigger
func SetTimeSeriesNames(request *http.Request, timeSeriesNames map[string]bool) {
	ctx := context.WithValue(request.Context(), timeSeriesNamesKey, timeSeriesNames)
	*request = *request.WithContext(ctx)
}

// GetTimeSeriesNames gets from requests context timeSeriesNames from saved trigger
func GetTimeSeriesNames(request *http.Request) map[string]bool {
	return request.Context().Value(timeSeriesNamesKey).(map[string]bool)
}

// GetRemoteConfig gets remote config from request context
func GetRemoteConfig(request *http.Request) *remote.Config {
	return request.Context().Value(remoteConfigKey).(*remote.Config)
}
