package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateAlert(t *testing.T) {

	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// Test request parameters
		if req.URL.String() != "/key" {
			t.Errorf(`Expected: "/key"\nActual: %v`, req.URL.String())
		}
		// Send response to be tested
		res.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := NewClient(server.URL, server.Client())

	Convey("CreateAlert Tests", t, func() {
		Convey("MessageType empty", func() {
			err := client.CreateAlert("key", CreateAlertRequest{MessageType: ""})
			So(err, ShouldResemble, fmt.Errorf("field MessageType cannot be empty"))
		})

		Convey("Not 200 OK response", func() {
			// Start a test server
			server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				// Test request parameters
				if req.URL.String() != "/key" {
					t.Errorf(`Expected: "/key"\nActual: %v`, req.URL.String())
				}
				res.WriteHeader(http.StatusInternalServerError)
				res.Header().Set("Content-Type", "application/json")
				json, _ := json.Marshal(map[string]string{
					"error": "test error",
				})
				res.Write(json)
				// Send response to be tested
			}))
			defer server.Close()
			client := NewClient(server.URL, server.Client())
			err := client.CreateAlert("key", CreateAlertRequest{MessageType: Critical})
			So(err, ShouldResemble, fmt.Errorf("victorops API request resulted in error with status %v: %v", http.StatusInternalServerError, `{"error":"test error"}`))
		})
		Convey("200 OK", func() {
			err := client.CreateAlert("key", CreateAlertRequest{MessageType: Critical})
			So(err, ShouldResemble, nil)
		})
	})
}
