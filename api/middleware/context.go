package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/remote"
)

// DatabaseContext sets to requests context configured database
func DatabaseContext(database moira.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), databaseKey, database)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// UserContext get x-webauth-user header and sets it in request context, if header is empty sets empty string
func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userLogin := request.Header.Get("x-webauth-user")
		ctx := context.WithValue(request.Context(), loginKey, userLogin)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// TriggerContext gets triggerId from parsed URI corresponding to trigger routes and set it to request context
func TriggerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		triggerID := chi.URLParam(request, "triggerId")
		if triggerID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("TriggerID must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), triggerIDKey, triggerID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// ContactContext gets contactID from parsed URI corresponding to trigger routes and set it to request context
func ContactContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := chi.URLParam(request, "contactId")
		if contactID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("ContactID must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), contactIDKey, contactID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// TagContext gets tagName from parsed URI corresponding to tag routes and set it to request context
func TagContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tag := chi.URLParam(request, "tag")
		if tag == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Tag must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), tagKey, tag)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// SubscriptionContext gets subscriptionId from parsed URI corresponding to subscription routes and set it to request context
func SubscriptionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		subscriptionID := chi.URLParam(request, "subscriptionId")
		if subscriptionID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("SubscriptionId must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionIDKey, subscriptionID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// RemoteConfigContext adds remote config struct to request context
func RemoteConfigContext(cfg *remote.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), remoteConfigKey, cfg)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// Paginate gets page and size values from URI query and set it to request context. If query has not values sets given values
func Paginate(defaultPage, defaultSize int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			page, err := strconv.ParseInt(request.URL.Query().Get("p"), 10, 64)
			if err != nil {
				page = defaultPage
			}
			size, err := strconv.ParseInt(request.URL.Query().Get("size"), 10, 64)
			if err != nil {
				size = defaultSize
			}

			ctxPage := context.WithValue(request.Context(), pageKey, page)
			ctxSize := context.WithValue(ctxPage, sizeKey, size)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}

// DateRange gets from and to values from URI query and set it to request context. If query has not values sets given values
func DateRange(defaultFrom, defaultTo string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			from := request.URL.Query().Get("from")
			if from == "" {
				from = defaultFrom
			}
			to := request.URL.Query().Get("to")
			if to == "" {
				to = defaultTo
			}

			ctxPage := context.WithValue(request.Context(), fromKey, from)
			ctxSize := context.WithValue(ctxPage, toKey, to)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}
