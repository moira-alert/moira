package handler

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
	"strconv"
)

func userContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userLogin := request.Header.Get("x-webauth-user")
		ctx := context.WithValue(request.Context(), "login", userLogin)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func triggerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		triggerId := chi.URLParam(request, "triggerId")
		if triggerId == "" {
			render.Render(writer, request, dto.ErrorInvalidRequest(fmt.Errorf("TriggerId must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), "triggerId", triggerId)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func tagContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tag := chi.URLParam(request, "tag")
		if tag == "" {
			render.Render(writer, request, dto.ErrorInvalidRequest(fmt.Errorf("Tag must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), "tag", tag)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func subscriptionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		triggerId := chi.URLParam(request, "subscriptionId")
		if triggerId == "" {
			render.Render(writer, request, dto.ErrorInvalidRequest(fmt.Errorf("SubscriptionId must be set")))
			return
		}
		ctx := context.WithValue(request.Context(), "subscriptionId", triggerId)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func paginate(defaultPage, defaultSize int64) func(next http.Handler) http.Handler {
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

			ctxPage := context.WithValue(request.Context(), "page", page)
			ctxSize := context.WithValue(ctxPage, "size", size)
			next.ServeHTTP(writer, request.WithContext(ctxSize))
		})
	}
}
