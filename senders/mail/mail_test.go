package mail

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFillSettings(t *testing.T) {
	Convey("Empty map", t, func() {
		sender := Sender{}
		err := sender.fillSettings(map[string]interface{}{}, nil, nil, "")
		So(err, ShouldResemble, fmt.Errorf("mail_from can't be empty"))
		So(sender, ShouldResemble, Sender{})
	})

	Convey("Has From", t, func() {
		sender := Sender{}
		settings := map[string]interface{}{"mail_from": "123"}
		Convey("No username", func() {
			err := sender.fillSettings(settings, nil, nil, "")
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{From: "123", Username: "123"})
		})
		Convey("Has username", func() {
			settings["smtp_user"] = "user"
			err := sender.fillSettings(settings, nil, nil, "")
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{From: "123", Username: "user"})
		})
	})
}

func TestParseTemplate(t *testing.T) {
	Convey("Template path is empty", t, func() {
		name, t, err := parseTemplate("")
		So(name, ShouldResemble, "mail")
		So(t, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})

	Convey("Template path no empty", t, func() {
		name, t, err := parseTemplate("bin/template")
		So(name, ShouldResemble, "template")
		So(t, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}
