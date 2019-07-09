package discord

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetResponse(t *testing.T) {
	Convey("getResponse tests", t, func() {
		sess := &discordgo.Session{}
		Convey("Empty map", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("cannot read the discord token from the config"))
			So(sender, ShouldResemble, Sender{DataBase: &MockDB{}})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"token":     "123",
				"front_uri": "http://moira.uri",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.session.Token, ShouldResemble, "Bot 123")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})
	})
}
