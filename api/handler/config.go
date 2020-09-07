package handler

import "net/http"

func getWebConfig(configContent []byte) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(configContent) //nolint
	})
}
