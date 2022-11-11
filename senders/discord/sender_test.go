package discord

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

type MockDB struct {
	moira.Database
}
type MockLock struct {
	moira.Lock
}

func (lock *MockLock) Acquire(<-chan struct{}) (lost <-chan struct{}, error error) {
	return lost, nil
}
func (db *MockDB) NewLock(string, time.Duration) moira.Lock {
	return &MockLock{}
}

func TestNewSender(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		Convey("Empty map", func() {
			sender, err := NewSender(map[string]string{}, logger, nil, &MockDB{})
			So(err, ShouldResemble, fmt.Errorf("cannot read the discord token from the config"))
			So(sender, ShouldBeNil)
		})

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"token":     "123",
				"front_uri": "http://moira.uri",
			}
			sender, err := NewSender(senderSettings, logger, location, &MockDB{})
			So(err, ShouldBeNil)
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.session.Token, ShouldResemble, "Bot 123")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})
	})
}
