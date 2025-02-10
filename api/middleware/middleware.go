package middleware

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// ContextKey used as key of api request context values.
type ContextKey string

func (key ContextKey) String() string {
	return "api context key " + string(key)
}

var (
	databaseKey          ContextKey = "database"
	searcherKey          ContextKey = "searcher"
	contactsTemplateKey  ContextKey = "contactsTemplate"
	triggerIDKey         ContextKey = "triggerID"
	clustersMetricTTLKey ContextKey = "clustersMetricTTL"
	populateKey          ContextKey = "populated"
	contactIDKey         ContextKey = "contactID"
	tagKey               ContextKey = "tag"
	subscriptionIDKey    ContextKey = "subscriptionID"
	pageKey              ContextKey = "page"
	sizeKey              ContextKey = "size"
	pagerIDKey           ContextKey = "pagerID"
	createPagerKey       ContextKey = "createPager"
	fromKey              ContextKey = "from"
	toKey                ContextKey = "to"
	loginKey             ContextKey = "login"
	timeSeriesNamesKey   ContextKey = "timeSeriesNames"
	metricSourceProvider ContextKey = "metricSourceProvider"
	targetNameKey        ContextKey = "target"
	teamIDKey            ContextKey = "teamID"
	teamUserIDKey        ContextKey = "teamUserIDKey"
	authKey              ContextKey = "auth"
	metricContextKey     ContextKey = "metric"
	statesContextKey     ContextKey = "states"
	limitsContextKey     ContextKey = "limits"
	searchTextContextKey ContextKey = "searchText"
	sortOrderContextKey  ContextKey = "sort"

	anonymousUser = "anonymous"
)

// GetDatabase gets moira.Database realization from request context.
func GetDatabase(request *http.Request) moira.Database {
	return request.Context().Value(databaseKey).(moira.Database)
}

// GetContactsTemplate gets contacts template from request context.
func GetContactsTemplate(request *http.Request) []api.WebContact {
	return request.Context().Value(contactsTemplateKey).([]api.WebContact)
}

// GetLogin gets user login string from request context, which was sets in UserContext middleware.
func GetLogin(request *http.Request) string {
	if request.Context() != nil && request.Context().Value(loginKey) != nil {
		if login := request.Context().Value(loginKey).(string); login != "" {
			return login
		}
	}

	return anonymousUser
}

// GetTriggerID gets TriggerID string from request context, which was sets in TriggerContext middleware.
func GetTriggerID(request *http.Request) string {
	return request.Context().Value(triggerIDKey).(string)
}

// GetMetricTTL gets local metric ttl duration time from request context, which was sets in TriggerContext middleware.
func GetMetricTTL(request *http.Request) map[moira.ClusterKey]time.Duration {
	return request.Context().Value(clustersMetricTTLKey).(map[moira.ClusterKey]time.Duration)
}

// GetPopulated get populate bool from request context, which was sets in TriggerContext middleware.
func GetPopulated(request *http.Request) bool {
	return request.Context().Value(populateKey).(bool)
}

// GetTag gets tag string from request context, which was sets in TagContext middleware.
func GetTag(request *http.Request) string {
	return request.Context().Value(tagKey).(string)
}

// GetSubscriptionID gets subscriptionId string from request context, which was sets in SubscriptionContext middleware.
func GetSubscriptionID(request *http.Request) string {
	return request.Context().Value(subscriptionIDKey).(string)
}

// GetContactID gets ContactID string from request context, which was sets in TriggerContext middleware.
func GetContactID(request *http.Request) string {
	return request.Context().Value(contactIDKey).(string)
}

// GetPage gets page value from request context, which was sets in Paginate middleware.
func GetPage(request *http.Request) int64 {
	return request.Context().Value(pageKey).(int64)
}

// GetSize gets size value from request context, which was sets in Paginate middleware.
func GetSize(request *http.Request) int64 {
	return request.Context().Value(sizeKey).(int64)
}

// GetPagerID is a function that gets pagerID value from request context, which was sets in Pager middleware.
func GetPagerID(request *http.Request) string {
	return request.Context().Value(pagerIDKey).(string)
}

// GetCreatePager is a function that gets createPager value from request context, which was sets in Pager middleware.
func GetCreatePager(request *http.Request) bool {
	return request.Context().Value(createPagerKey).(bool)
}

// GetFromStr gets 'from' value from request context, which was sets in DateRange middleware.
func GetFromStr(request *http.Request) string {
	return request.Context().Value(fromKey).(string)
}

// GetToStr gets 'to' value from request context, which was sets in DateRange middleware.
func GetToStr(request *http.Request) string {
	return request.Context().Value(toKey).(string)
}

// SetTimeSeriesNames sets to request's context timeSeriesNames from saved trigger.
func SetTimeSeriesNames(request *http.Request, timeSeriesNames map[string]bool) {
	ctx := context.WithValue(request.Context(), timeSeriesNamesKey, timeSeriesNames)
	*request = *request.WithContext(ctx)
}

// GetTimeSeriesNames gets from request's context timeSeriesNames from saved trigger.
func GetTimeSeriesNames(request *http.Request) map[string]bool {
	return request.Context().Value(timeSeriesNamesKey).(map[string]bool)
}

// GetTriggerTargetsSourceProvider gets trigger targets source provider.
func GetTriggerTargetsSourceProvider(request *http.Request) *metricSource.SourceProvider {
	return request.Context().Value(metricSourceProvider).(*metricSource.SourceProvider)
}

// GetTargetName gets target name.
func GetTargetName(request *http.Request) string {
	return request.Context().Value(targetNameKey).(string)
}

// GetTeamID gets team id.
func GetTeamID(request *http.Request) string {
	teamID := request.Context().Value(teamIDKey)
	if teamID == nil {
		return ""
	}
	return teamID.(string)
}

// GetTeamUserID gets team user id.
func GetTeamUserID(request *http.Request) string {
	return request.Context().Value(teamUserIDKey).(string)
}

// SetContextValueForTest is a helper function that is needed for testing purposes and sets context values with local ContextKey type.
func SetContextValueForTest(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, ContextKey(key), value)
}

// GetAuth gets authorization configuration.
func GetAuth(request *http.Request) *api.Authorization {
	return request.Context().Value(authKey).(*api.Authorization)
}

// GetMetric is used to retrieve metric name.
func GetMetric(request *http.Request) string {
	return request.Context().Value(metricContextKey).(string)
}

// GetStates is used to retrieve trigger state.
func GetStates(request *http.Request) map[string]struct{} {
	return request.Context().Value(statesContextKey).(map[string]struct{})
}

// GetLimits returns configured limits.
func GetLimits(request *http.Request) api.LimitsConfig {
	return request.Context().Value(limitsContextKey).(api.LimitsConfig)
}

// GetSearchText returns search text regexp.
func GetSearchText(request *http.Request) *regexp.Regexp {
	return request.Context().Value(searchTextContextKey).(*regexp.Regexp)
}

// GetSortOrder returns api.SortOrder.
func GetSortOrder(request *http.Request) api.SortOrder {
	return request.Context().Value(sortOrderContextKey).(api.SortOrder)
}
