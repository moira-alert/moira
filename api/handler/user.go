package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
	"os"
)

func user(router chi.Router) {
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		if err := render.Render(writer, request, &dto.User{request.Header.Get("login")}); err != nil {
			return
		}
	})

	router.Get("/settings", func(writer http.ResponseWriter, request *http.Request) {
		//todo не забыть пропихнуть user в каждый subscription

		userLogin := request.Header.Get("login")
		fmt.Fprintf(os.Stderr, "%s", userLogin)
		userSettings := dto.UserSettings{
			User:          dto.User{Login: userLogin},
			Contacts:      make([]moira.ContactData, 0),
			Subscriptions: make([]moira.SubscriptionData, 0),
		}

		subscriptionIds, err := database.GetUserSubscriptions(userLogin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err.Error())
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		contactIds, err := database.GetUserContacts(userLogin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err.Error())
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		for _, id := range subscriptionIds {
			subscription, err := database.GetSubscription(id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			userSettings.Subscriptions = append(userSettings.Subscriptions, subscription)
		}

		for _, id := range contactIds {
			contact, err := database.GetContact(id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			userSettings.Contacts = append(userSettings.Contacts, contact)
		}
		if err := render.Render(writer, request, &userSettings); err != nil {
			return
		}
	})
}
