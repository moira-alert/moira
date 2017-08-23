package middleware

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"net/http"
	"strconv"
)

func DatabaseContext(database moira.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), databaseKey, database)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userLogin := request.Header.Get("x-webauth-user")
		ctx := context.WithValue(request.Context(), loginKey, userLogin)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

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

func SubscriptionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		triggerID := chi.URLParam(request, "subscriptionId")
		if triggerID == "" {
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("SubscriptionId must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), subscriptionIDKey, triggerID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

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
