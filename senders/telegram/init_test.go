package telegram

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
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

func Test_customOnErrorFunc(t *testing.T) {
	Convey("test customOnErrorFunc hides credential and logs", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		logger := mock_moira_alert.NewMockLogger(mockCtrl)
		eventsBuilder := mock_moira_alert.NewMockEventBuilder(mockCtrl)

		sender := Sender{
			logger:   logger,
			apiToken: "1111111111:SecretTokenabc_987654321hellokonturmoira",
		}

		err := fmt.Errorf("https://some.api.of.telegram/bot%s/update failed to update", sender.apiToken)

		logger.EXPECT().Warning().Return(eventsBuilder).AnyTimes()
		eventsBuilder.EXPECT().Error(errors.New(strings.ReplaceAll(err.Error(), sender.apiToken, hidden))).Return(eventsBuilder)
		eventsBuilder.EXPECT().Msg(errorInsideTelebotMsg)

		sender.customOnErrorFunc(err, nil)
	})
}
