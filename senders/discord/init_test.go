package discord

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

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
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{DataBase: &MockDB{}}
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
