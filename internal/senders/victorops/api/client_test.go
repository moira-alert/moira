package api

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewClient(t *testing.T) {
	Convey("NewClient tests", t, func() {
		Convey("Nil http client", func() {
			client := NewClient("https://testurl.com", nil)
			So(client, ShouldResemble, &Client{httpClient: http.DefaultClient, routingURL: "https://testurl.com"})
		})

		Convey("Custom http client", func() {
			client := NewClient("https://testurl.com", http.DefaultClient)
			So(client, ShouldResemble, &Client{httpClient: http.DefaultClient, routingURL: "https://testurl.com"})
		})
	})
}
