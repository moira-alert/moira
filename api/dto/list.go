package dto

import (
	"net/http"

	"github.com/go-chi/render"
)

// ListDTO is a generic struct to create list types dto.
type ListDTO[T render.Renderer] struct {
	// List of entities.
	List []T `json:"list"`
	// Page number.
	Page int64 `json:"page" example:"0" format:"int64"`
	// Size is the amount of entities per Page.
	Size int64 `json:"size" example:"100" format:"int64"`
	// Total amount of entities in the database.
	Total int64 `json:"total" example:"10" format:"int64"`
}

func (*ListDTO[T]) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
