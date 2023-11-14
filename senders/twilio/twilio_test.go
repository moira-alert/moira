package twilio

import (
	"fmt"
	"testing"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Tests init twilio sender", t, func() {
		sender := Sender{}
		settings := map[string]interface{}{}
		Convey("no api asid", func() {
			err := sender.Init(settings, logger, nil, "15:04", dataBase)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_sid param from config", ""))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", func() {
			err := sender.Init(settings, logger, nil, "15:04", dataBase)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_authtoken param from config", ""))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", func() {
			err := sender.Init(settings, logger, nil, "15:04", dataBase)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_fromphone param from config", ""))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", func() {
			err := sender.Init(settings, logger, nil, "15:04", dataBase)
			So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", ""))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("config sms", func() {
			settings["type"] = "twilio sms"
			err := sender.Init(settings, logger, location, "15:04", dataBase)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{sender: &twilioSenderSms{
				twilioSender{
					client:       twilio_client.NewClient("123", "321"),
					APIFromPhone: "12345678989",
					logger:       logger,
					location:     location,
				},
			}})
		})

		Convey("config voice", func() {
			settings["type"] = "twilio voice"
			Convey("no voice url", func() {
				err := sender.Init(settings, logger, location, "15:04", dataBase)
				So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				So(sender, ShouldResemble, Sender{})
			})

			Convey("has voice url", func() {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", func() {
					settings["append_message"] = true
					err := sender.Init(settings, logger, location, "15:04", dataBase)
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio_client.NewClient("123", "321"),
							APIFromPhone: "12345678989",
							logger:       logger,
							location:     location,
						},
						voiceURL:      "url here",
						appendMessage: true,
					}})
				})

				Convey("append_message is false", func() {
					settings["append_message"] = false
					err := sender.Init(settings, logger, location, "15:04", dataBase)
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio_client.NewClient("123", "321"),
							APIFromPhone: "12345678989",
							logger:       logger,
							location:     location,
						},
						voiceURL:      "url here",
						appendMessage: false,
					}})
				})
			})
		})
	})
}
