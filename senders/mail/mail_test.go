package mail

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const mailType = "mail"

func TestFillSettings(t *testing.T) {
	Convey("Empty map", t, func() {
		client := mailClient{}
		senderIdent, err := client.fillSettings(map[string]interface{}{}, nil, nil, "")
		So(err, ShouldResemble, fmt.Errorf("mail_from can't be empty"))
		So(client, ShouldResemble, mailClient{})
		So(senderIdent, ShouldEqual, "")
	})

	Convey("Has From", t, func() {
		client := mailClient{}
		settings := map[string]interface{}{
			"type":      mailType,
			"mail_from": "123",
		}

		Convey("No username", func() {
			senderIdent, err := client.fillSettings(settings, nil, nil, "")
			So(err, ShouldBeNil)
			So(client, ShouldResemble, mailClient{
				From:     "123",
				Username: "123",
			})
			So(senderIdent, ShouldEqual, settings["type"])
		})

		Convey("Has username", func() {
			settings["smtp_user"] = "user"
			senderIdent, err := client.fillSettings(settings, nil, nil, "")
			So(err, ShouldBeNil)
			So(client, ShouldResemble, mailClient{
				From:     "123",
				Username: "user",
			})
			So(senderIdent, ShouldEqual, settings["type"])
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
