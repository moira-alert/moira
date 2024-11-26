package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// DatabaseContext sets to requests context configured database.
func DatabaseContext(database moira.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), databaseKey, database)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// SearchIndexContext sets to requests context configured moira.index.searchIndex.
func SearchIndexContext(searcher moira.Searcher) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), searcherKey, searcher)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// ContactsTemplateContext sets to requests context contacts template.
func ContactsTemplateContext(contactsTemplate []api.WebContact) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), contactsTemplateKey, contactsTemplate)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// UserContext get x-webauth-user header and sets it in request context, if header is empty sets empty string.
func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userLogin := request.Header.Get("x-webauth-user")
		ctx := context.WithValue(request.Context(), loginKey, userLogin)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// TriggerContext gets triggerId from parsed URI corresponding to trigger routes and set it to request context.
func TriggerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		triggerID := chi.URLParam(request, "triggerId")
		if triggerID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("triggerID must be set"))) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), triggerIDKey, triggerID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// ContactContext gets contactID from parsed URI corresponding to trigger routes and set it to request context.
func ContactContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := chi.URLParam(request, "contactId")
		if contactID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("contactID must be set"))) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), contactIDKey, contactID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// TagContext gets tagName from parsed URI corresponding to tag routes and set it to request context.
func TagContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tag := chi.URLParam(request, "tag")
		if tag == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("tag must be set"))) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), tagKey, tag)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// SubscriptionContext gets subscriptionId from parsed URI corresponding to subscription routes and set it to request context.
func SubscriptionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		subscriptionID := chi.URLParam(request, "subscriptionId")
		if subscriptionID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("subscriptionId must be set"))) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionIDKey, subscriptionID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// MetricSourceProvider adds metrics source provider to context.
func MetricSourceProvider(sourceProvider *metricSource.SourceProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), metricSourceProvider, sourceProvider)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// Paginate gets page and size values from URI query and set it to request context. If query has not values sets given values.
func Paginate(defaultPage, defaultSize int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			page, err := strconv.ParseInt(urlValues.Get("p"), 10, 64)
			if err != nil {
				page = defaultPage
			}

			size, err := strconv.ParseInt(urlValues.Get("size"), 10, 64)
			if err != nil {
				size = defaultSize
			}

			ctxPage := context.WithValue(request.Context(), pageKey, page)
			ctxSize := context.WithValue(ctxPage, sizeKey, size)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}

// Pager is a function that takes pager id from query.
func Pager(defaultCreatePager bool, defaultPagerID string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			pagerID := urlValues.Get("pagerID")
			if pagerID == "" {
				pagerID = defaultPagerID
			}

			createPager, err := strconv.ParseBool(urlValues.Get("createPager"))
			if err != nil {
				createPager = defaultCreatePager
			}

			ctxPager := context.WithValue(request.Context(), pagerIDKey, pagerID)
			ctxSize := context.WithValue(ctxPager, createPagerKey, createPager)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}

// Populate gets bool value populate from URI query and set it to request context. If query has not values sets given values.
func Populate(defaultPopulated bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			populate, err := strconv.ParseBool(urlValues.Get("populated"))
			if err != nil {
				populate = defaultPopulated
			}

			ctxTemplate := context.WithValue(request.Context(), populateKey, populate)
			next.ServeHTTP(writer, request.WithContext(ctxTemplate))
		})
	}
}

// Triggers gets string value target from URI query and set it to request context. If query has not values sets given values.
func Triggers(metricTTL map[moira.ClusterKey]time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()

			ctx = context.WithValue(ctx, clustersMetricTTLKey, metricTTL)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// DateRange gets from and to values from URI query and set it to request context. If query has not values sets given values.
func DateRange(defaultFrom, defaultTo string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			from := urlValues.Get("from")
			if from == "" {
				from = defaultFrom
			}

			to := urlValues.Get("to")
			if to == "" {
				to = defaultTo
			}

			ctxPage := context.WithValue(request.Context(), fromKey, from)
			ctxSize := context.WithValue(ctxPage, toKey, to)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}

// TargetName is a function that gets target name value from query string and places it in context. If query does not have value sets given value.
func TargetName(defaultTargetName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			targetName := urlValues.Get("target")
			if targetName == "" {
				targetName = defaultTargetName
			}

			ctx := context.WithValue(request.Context(), targetNameKey, targetName)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// TeamContext gets teamId from parsed URI corresponding to team routes and set it to request context.
func TeamContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		teamID := chi.URLParam(request, "teamId")
		if teamID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("teamId must be set"))) //nolint:errcheck
			return
		}
		ctx := context.WithValue(request.Context(), teamIDKey, teamID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// TeamUserIDContext gets userId from parsed URI corresponding to team routes and set it to request context.
func TeamUserIDContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userID := chi.URLParam(request, "teamUserId")
		if userID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("userId must be set"))) //nolint:errcheck
			return
		}
		ctx := context.WithValue(request.Context(), teamUserIDKey, userID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// AuthorizationContext sets given authorization configuration to request context.
func AuthorizationContext(auth *api.Authorization) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), authKey, auth)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// MetricContext is a function that gets `metric` value from query string and places it in context. If query does not have value sets given value.
func MetricContext(defaultMetric string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			metric := urlValues.Get("metric")
			if metric == "" {
				metric = defaultMetric
			}

			ctx := context.WithValue(request.Context(), metricContextKey, metric)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

const statesArraySeparator = ","

// StatesContext is a function that gets `states` value from query string and places it in context. If query does not have value empty map will be used.
func StatesContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			states := make(map[string]struct{}, 0)

			statesStr := urlValues.Get("states")
			if statesStr != "" {
				statesList := strings.Split(statesStr, statesArraySeparator)
				for _, state := range statesList {
					if !moira.State(state).IsValid() {
						_ = render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("bad state in query parameter: %s", state)))
						return
					}
					states[state] = struct{}{}
				}
			}

			ctx := context.WithValue(request.Context(), statesContextKey, states)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// LimitsContext places api.LimitsConfig to request context.
func LimitsContext(limit api.LimitsConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), limitsContextKey, limit)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// SearchTextContext compiles and puts search text regex to request context.
func SearchTextContext(defaultRegex *regexp.Regexp) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			var searchTextRegex *regexp.Regexp

			searchText := urlValues.Get("searchText")
			if searchText != "" {
				searchTextRegex, err = regexp.Compile(searchText)
				if err != nil {
					render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("failed to parse searchText template '%s': %w", searchText, err))) //nolint
					return
				}
			} else {
				searchTextRegex = defaultRegex
			}

			ctx := context.WithValue(request.Context(), searchTextContextKey, searchTextRegex)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// SortOrderContext puts sort order to request context.
func SortOrderContext(defaultSortOrder api.SortOrder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			urlValues, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
				return
			}

			queryParamName := "sort"

			var sortOrder api.SortOrder
			if !urlValues.Has(queryParamName) {
				sortOrder = defaultSortOrder
			} else {
				sortVal := api.SortOrder(urlValues.Get(queryParamName))
				switch sortVal {
				case api.NoSortOrder, api.AscSortOrder, api.DescSortOrder:
					sortOrder = sortVal
				default:
					sortOrder = defaultSortOrder
				}
			}

			ctx := context.WithValue(request.Context(), sortOrderContextKey, sortOrder)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
