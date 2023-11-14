package discord

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const discordType = "discord"

type MockDB struct {
	moira.Database
}

type MockLock struct {
	moira.Lock
}

func (lock *MockLock) Acquire(stop <-chan struct{}) (lost <-chan struct{}, error error) {
	return lost, nil
}

func (db *MockDB) NewLock(name string, ttl time.Duration) moira.Lock {
	return &MockLock{}
}

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	database := &MockDB{}

	Convey("Init tests", t, func() {
		sender := Sender{}

		Convey("Empty map", func() {
			err := sender.Init(map[string]interface{}{}, logger, nil, "", database)
			So(err, ShouldResemble, fmt.Errorf("cannot read the discord token from the config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]interface{}{
				"type":      discordType,
				"token":     "123",
				"front_uri": "http://moira.uri",
			}

			sender.Init(senderSettings, logger, location, "15:04", database) //nolint
			So(sender.clients, ShouldHaveLength, 1)
			client := sender.clients[discordType]

			So(client.frontURI, ShouldResemble, "http://moira.uri")
			So(client.session.Token, ShouldResemble, "Bot 123")
			So(client.logger, ShouldResemble, logger)
			So(client.location, ShouldResemble, location)
		})

		Convey("Multiple init", func() {
			senderSettings1 := map[string]interface{}{
				"type":      discordType,
				"name":      "discord1",
				"token":     "123",
				"front_uri": "http://moira.uri",
			}

			err := sender.Init(senderSettings1, logger, location, "15:04", database)
			So(err, ShouldBeNil)
			So(sender.clients, ShouldHaveLength, 1)

			client1 := sender.clients["discord1"]
			So(client1.frontURI, ShouldResemble, "http://moira.uri")
			So(client1.session.Token, ShouldResemble, "Bot 123")
			So(client1.logger, ShouldResemble, logger)
			So(client1.location, ShouldResemble, location)

			senderSettings2 := map[string]interface{}{
				"type":      discordType,
				"name":      "discord2",
				"token":     "456",
				"front_uri": "http://moira.uri",
			}

			err = sender.Init(senderSettings2, logger, location, "15:04", database)
			So(err, ShouldBeNil)

			So(sender.clients, ShouldHaveLength, 2)
			client2 := sender.clients["discord2"]

			So(client2.frontURI, ShouldResemble, "http://moira.uri")
			So(client2.session.Token, ShouldResemble, "Bot 456")
			So(client2.logger, ShouldResemble, logger)
			So(client2.location, ShouldResemble, location)
		})
	})
}
