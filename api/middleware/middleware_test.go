package middleware

import (
	"context"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetLogin(t *testing.T) {
	Convey("Request does not contain login, should get anonymous", t, func() {
		req, err := http.NewRequest(http.MethodGet, "https://testurl.com", http.NoBody)
		So(err, ShouldBeNil)
		So(anonymousUser, ShouldEqual, GetLogin(req))
	})

	Convey("Request contains login, but empty, should get anonymous", t, func() {
		req, err := http.NewRequest(http.MethodGet, "https://testurl.com", http.NoBody)
		ctx := context.WithValue(req.Context(), loginKey, "")
		req = req.WithContext(ctx)
		So(err, ShouldBeNil)
		So(anonymousUser, ShouldEqual, GetLogin(req))
	})

	Convey("Request contains login header, should get that", t, func() {
		req, err := http.NewRequest(http.MethodGet, "https://testurl.com", http.NoBody)
		ctx := context.WithValue(req.Context(), loginKey, "awesome_user")
		req = req.WithContext(ctx)
		So(err, ShouldBeNil)
		So("awesome_user", ShouldEqual, GetLogin(req))
	})
}
