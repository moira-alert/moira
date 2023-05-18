package handler

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(contactFilter)
		router.Get("/", getContactById)
		router.Get("/events", getContactByIdWithEvents)
		router.Put("/", updateContact)
		router.Delete("/", removeContact)
		router.Post("/test", sendTestContactNotification)
	})
}

func getAllContacts(writer http.ResponseWriter, request *http.Request) {
	contacts, err := controller.GetAllContacts(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func getContactById(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	contact, err := controller.GetContactById(database, contactData.ID)

	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getContactByIdWithEvents(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	params := request.URL.Query()

	// TODO: make this checks as the middleware and pass to retrive function
	if params.Get("from") != "" && params.Get("to") != "" {

		contact, err := controller.GetContactByIdWithEvents(database, contactData.ID)

		if err != nil {
			render.Render(writer, request, err)
			return
		}

		if err := render.Render(writer, request, contact); err != nil {
			render.Render(writer, request, api.ErrorRender(err))
			return
		}
	} else {
		render.Render(
			writer,
			request,
			api.ErrorInvalidRequest(fmt.Errorf("You have to specify either 'from' and 'to' request params")))
	}
}

func createNewContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	userLogin := middleware.GetLogin(request)

	if err := controller.CreateContact(database, contact, userLogin, ""); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// contactFilter is middleware for check contact existence and user permissions
func contactFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := middleware.GetContactID(request)
		userLogin := middleware.GetLogin(request)
		contactData, err := controller.CheckUserPermissionsForContact(database, contactID, userLogin)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}
		ctx := context.WithValue(request.Context(), contactKey, contactData)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func updateContact(writer http.ResponseWriter, request *http.Request) {
	contactDTO := dto.Contact{}
	if err := render.Bind(request, &contactDTO); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	contactData := request.Context().Value(contactKey).(moira.ContactData)

	contactDTO, err := controller.UpdateContact(database, contactDTO, contactData)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, &contactDTO); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

func removeContact(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	err := controller.RemoveContact(database, contactData.ID, contactData.User, "")
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

func sendTestContactNotification(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)
	err := controller.SendTestContactNotification(database, contactID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}
