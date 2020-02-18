package docs

import "github.com/moira-alert/moira/api"

// swagger:response errorResponse
type errorResponse struct {
	// in:body
	Body api.ErrorResponse
}

type contactIdContext struct {
	ContactId string `json:"contactId"`
}
