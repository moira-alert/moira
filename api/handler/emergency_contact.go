package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func emergencyContact(router chi.Router) {
	router.With(middleware.AdminOnlyMiddleware()).Get("/", getEmergencyContacts)
	router.Post("/", createEmergencyContact)
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(emergencyContactFilter)
		router.Use(contactFilter)
		router.Get("/", getEmergencyContactByID)
		router.Put("/", updateEmergencyContact)
		router.Delete("/", removeEmergencyContact)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Gets all Moira emergency contacts
//	@id			get-all-emergency-contacts
//	@tags		emergency-contact
//	@produce	json
//	@success	200	{object}	dto.EmergencyContactList		"Contacts fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/emergency-contact [get]
func getEmergencyContacts(writer http.ResponseWriter, request *http.Request) {
	emergencyContacts, err := controller.GetEmergencyContacts(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, emergencyContacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get emergency contact by it's contact ID
//	@id			get-emergency-contact-by-id
//	@tags		emergency-contact
//	@produce	json
//	@param		contactID	path		string							true	"Contact ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.EmergencyContact			"Successfully received contact"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/emergency-contact/{contactID} [get]
func getEmergencyContactByID(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)

	emergencyContact, err := controller.GetEmergencyContact(database, contactID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, emergencyContact); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Creates a new emergency contact for the current user
//	@id			create-emergency-contact
//	@tags		emergency-contact
//	@accept		json
//	@produce	json
//	@param		emergency-contact	body		dto.EmergencyContact				true	"Emergency contact data"
//	@success	200					{object}	dto.SaveEmergencyContactResponse	"Emergency contact created successfully"
//	@failure	400					{object}	api.ErrorInvalidRequestExample		"Bad request from client"
//	@failure	422					{object}	api.ErrorRenderExample				"Render error"
//	@failure	500					{object}	api.ErrorInternalServerExample		"Internal server error"
//	@router		/emergency-contact [post]
func createEmergencyContact(writer http.ResponseWriter, request *http.Request) {
	emergencyContactDTO := &dto.EmergencyContact{}
	if err := render.Bind(request, emergencyContactDTO); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	userLogin := middleware.GetLogin(request)
	auth := middleware.GetAuth(request)

	response, err := controller.CreateEmergencyContact(database, auth, emergencyContactDTO, userLogin)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Updates an existing contact to the values passed in the request body
//	@id			update-emergency-contact
//	@tags		emergency-contact
//	@accept		json
//	@produce	json
//	@param		contactID			path		string								true	"ID of the contact to update"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		emergency-contact	body		dto.EmergencyContact				true	"Updated emergency contact data"
//	@success	200					{object}	dto.SaveEmergencyContactResponse	"Updated emergency contact"
//	@failure	400					{object}	api.ErrorInvalidRequestExample		"Bad request from client"
//	@failure	403					{object}	api.ErrorForbiddenExample			"Forbidden"
//	@failure	404					{object}	api.ErrorNotFoundExample			"Resource not found"
//	@failure	422					{object}	api.ErrorRenderExample				"Render error"
//	@failure	500					{object}	api.ErrorInternalServerExample		"Internal server error"
//	@router		/emergency-contact/{contactID} [put]
func updateEmergencyContact(writer http.ResponseWriter, request *http.Request) {
	emergencyContactDTO := &dto.EmergencyContact{}
	if err := render.Bind(request, emergencyContactDTO); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	contactID := middleware.GetContactID(request)

	response, err := controller.UpdateEmergencyContact(database, contactID, emergencyContactDTO)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Deletes emergency contact for the current user
//	@id			remove-emergency-contact
//	@accept		json
//	@produce	json
//	@tags		emergency-contact
//	@param		contactID	path	string	true	"ID of the emergency contact to remove"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			"Emergency contact has been deleted"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/emergency-contact/{contactID} [delete]
func removeEmergencyContact(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)

	if err := controller.RemoveEmergencyContact(database, contactID); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
}

func emergencyContactFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contactID := middleware.GetContactID(request)

		if _, err := controller.GetEmergencyContact(database, contactID); err != nil {
			render.Render(writer, request, err) //nolint
			return
		}

		next.ServeHTTP(writer, request)
	})
}
