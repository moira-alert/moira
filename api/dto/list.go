package dto

import (
	"net/http"

	"github.com/go-chi/render"
)

type ListDto[T render.Renderer] struct {
	List  []T   `json:"list"`
	Page  int64 `json:"page" example:"0" format:"int64"`
	Size  int64 `json:"size" example:"100" format:"int64"`
	Total int64 `json:"total" example:"10" format:"int64"`
}

func (*ListDto[T]) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
