package telegram

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

const telegramType = "telegram"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Init tests", t, func() {
		sender := Sender{}
		Convey("Empty map", func() {
			opts := moira.InitOptions{
				SenderSettings: map[string]interface{}{},
				Logger:         logger,
				Location:       nil,
				Database:       dataBase,
			}

			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("can not read telegram api_token from config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]interface{}{
				"type":      telegramType,
				"api_token": "123",
				"front_uri": "http://moira.uri",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				Database:       dataBase,
			}

			err := sender.Init(opts)
			So(err, ShouldNotBeNil)
		})
	})
}
