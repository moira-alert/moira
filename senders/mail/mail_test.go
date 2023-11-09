package mail

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const mailType = "mail"

func TestFillSettings(t *testing.T) {
	Convey("Empty map", t, func() {
		sender := Sender{}
		sendersNameToType := make(map[string]string)

		err := sender.fillSettings(map[string]interface{}{}, nil, nil, "", sendersNameToType)
		So(err, ShouldResemble, fmt.Errorf("mail_from can't be empty"))
		So(sender, ShouldResemble, Sender{})
	})

	Convey("Has From", t, func() {
		sender := Sender{}
		settings := map[string]interface{}{
			"type":      mailType,
			"mail_from": "123",
		}

		Convey("No username", func() {
			sendersNameToType := make(map[string]string)

			err := sender.fillSettings(settings, nil, nil, "", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{From: "123", Username: "123"})
			So(sendersNameToType[mailType], ShouldEqual, settings["type"])
		})

		Convey("Has username", func() {
			settings["smtp_user"] = "user"
			sendersNameToType := make(map[string]string)

			err := sender.fillSettings(settings, nil, nil, "", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{From: "123", Username: "user"})
			So(sendersNameToType[mailType], ShouldEqual, settings["type"])
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
