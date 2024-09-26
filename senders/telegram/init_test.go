package telegram

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{}

		validatorErr := validator.ValidationErrors{}

		Convey("With empty api_token", func() {
			err := sender.Init(map[string]interface{}{}, logger, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]interface{}{
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}
			sender.Init(senderSettings, logger, location, "15:04") //nolint
			So(sender.logger, ShouldResemble, logger)
			So(sender.apiToken, ShouldResemble, "123")
		})
	})
}
