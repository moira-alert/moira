package handler

import (
	"net/http"
	"strconv"
)

func getPageAndSize(request *http.Request, defaultPage int64, defaultSize int64) (page, size int64) {
	page, err := strconv.ParseInt(request.URL.Query().Get("p"), 10, 64)
	if err != nil {
		page = 0
	}
	size, err = strconv.ParseInt(request.URL.Query().Get("size"), 10, 64)
	if err != nil {
		size = 10
	}
	return
}
