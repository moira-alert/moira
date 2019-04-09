package mail

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFillSettings(t *testing.T) {
	Convey("Empty map", t, func(c C) {
		sender := Sender{}
		err := sender.fillSettings(map[string]string{}, nil, nil, "")
		c.So(err, ShouldResemble, fmt.Errorf("mail_from can't be empty"))
		c.So(sender, ShouldResemble, Sender{})
	})

	Convey("Has From", t, func(c C) {
		sender := Sender{}
		settings := map[string]string{"mail_from": "123"}
		Convey("No username", t, func(c C) {
			err := sender.fillSettings(settings, nil, nil, "")
			c.So(err, ShouldBeNil)
			c.So(sender, ShouldResemble, Sender{From: "123", Username: "123"})
		})
		Convey("Has username", t, func(c C) {
			settings["smtp_user"] = "user"
			err := sender.fillSettings(settings, nil, nil, "")
			c.So(err, ShouldBeNil)
			c.So(sender, ShouldResemble, Sender{From: "123", Username: "user"})
		})
	})
}

func TestParseTemplate(t *testing.T) {
	Convey("Template path is empty", t, func(c C) {
		name, t, err := parseTemplate("")
		c.So(name, ShouldResemble, "mail")
		c.So(t, ShouldNotBeNil)
		c.So(err, ShouldBeNil)
	})

	Convey("Template path no empty", t, func(c C) {
		name, t, err := parseTemplate("bin/template")
		c.So(name, ShouldResemble, "template")
		c.So(t, ShouldBeNil)
		c.So(err, ShouldNotBeNil)
	})
}
