package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func subscription(router chi.Router) {
	router.Get("/", getUserSubscriptions)
	router.Put("/", createSubscription)
	router.Route("/{subscriptionId}", func(router chi.Router) {
		router.Use(subscriptionContext)
		router.Delete("/", deleteSubscription)
		router.Put("/test", sendTestNotification)
	})
}

func getUserSubscriptions(writer http.ResponseWriter, request *http.Request) {
	userLogin := request.Header.Get("login")
	if userLogin == "" {
		if err := render.Render(writer, request, dto.ErrorUserCanNotBeEmpty); err != nil {
			render.Render(writer, request, dto.ErrorRender(err))
		}
		return
	}

	contacts, err := controller.GetUserSubscriptions(database, userLogin)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func createSubscription(writer http.ResponseWriter, request *http.Request) {
	subscription := &dto.Subscription{}
	if err := render.Bind(request, subscription); err != nil {
		render.Render(writer, request, dto.ErrorInvalidRequest(err))
		return
	}
	userLogin := request.Header.Get("login")
	if userLogin == "" {
		render.Render(writer, request, dto.ErrorUserCanNotBeEmpty)
		return
	}

	if err := controller.WriteSubscription(database, userLogin, subscription); err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, subscription); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func deleteSubscription(writer http.ResponseWriter, request *http.Request) {
	userLogin := request.Header.Get("login")
	if userLogin == "" {
		render.Render(writer, request, dto.ErrorUserCanNotBeEmpty)
		return
	}
	subscriptionId := request.Context().Value("subscriptionId").(string)
	if err := controller.DeleteSubscription(database, subscriptionId, userLogin); err != nil {
		render.Render(writer, request, err)
	}
}

func sendTestNotification(writer http.ResponseWriter, request *http.Request) {
	subscriptionId := request.Context().Value("subscriptionId").(string)
	if err := controller.SendTestNotification(database, subscriptionId); err != nil {
		render.Render(writer, request, err)
	}
}
