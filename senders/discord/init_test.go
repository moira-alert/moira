package discord

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
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

func (lock *MockLock) Acquire(stop <-chan struct{}) (lost <-chan struct{}, err error) {
	return lost, nil
}

func (db *MockDB) NewLock(name string, ttl time.Duration) moira.Lock {
	return &MockLock{}
}

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")

	Convey("Init tests", t, func() {
		sender := Sender{DataBase: &MockDB{}}

		validatorErr := validator.ValidationErrors{}

		Convey("With empty token", func() {
			senderSettings := map[string]interface{}{}

			err := sender.Init(senderSettings, logger, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{DataBase: &MockDB{}})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]interface{}{
				"token":     "123",
				"front_uri": "http://moira.uri",
			}

			err := sender.Init(senderSettings, logger, location, "15:04") //nolint
			So(err, ShouldBeNil)
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.session.Token, ShouldResemble, "Bot 123")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})
	})
}
