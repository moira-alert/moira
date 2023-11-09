package twilio

import (
	"fmt"
	"testing"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const twilioType = "twilio"

func TestInit(t *testing.T) {
	Convey("Tests init twilio sender", t, func() {
		sender := Sender{}
		logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
		location, _ := time.LoadLocation("UTC")
		settings := map[string]interface{}{
			"type": twilioType,
		}

		Convey("no api asid", func() {
			sendersNameToType := make(map[string]string)

			err := sender.Init(settings, logger, nil, "15:04", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_sid param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", func() {
			sendersNameToType := make(map[string]string)

			err := sender.Init(settings, logger, nil, "15:04", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_authtoken param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", func() {
			sendersNameToType := make(map[string]string)

			err := sender.Init(settings, logger, nil, "15:04", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_fromphone param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", func() {
			sendersNameToType := make(map[string]string)

			err := sender.Init(settings, logger, nil, "15:04", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", twilioType))
			So(sender, ShouldResemble, Sender{})
			So(sendersNameToType[twilioType], ShouldEqual, settings["type"])
		})

		Convey("config sms", func() {
			sendersNameToType := make(map[string]string)
			settings["type"] = "twilio sms"

			err := sender.Init(settings, logger, location, "15:04", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{sender: &twilioSenderSms{
				twilioSender{
					client:       twilio_client.NewClient("123", "321"),
					APIFromPhone: "12345678989",
					logger:       logger,
					location:     location,
				},
			}})
			So(sendersNameToType["twilio sms"], ShouldEqual, settings["type"])
		})

		Convey("config voice", func() {
			settings["type"] = "twilio voice"
			Convey("no voice url", func() {
				sendersNameToType := make(map[string]string)

				err := sender.Init(settings, logger, location, "15:04", sendersNameToType)
				So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				So(sender, ShouldResemble, Sender{})
				So(sendersNameToType["twilio voice"], ShouldEqual, settings["type"])
			})

			Convey("has voice url", func() {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", func() {
					sendersNameToType := make(map[string]string)
					settings["append_message"] = true

					err := sender.Init(settings, logger, location, "15:04", sendersNameToType)
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
					So(sendersNameToType["twilio voice"], ShouldEqual, settings["type"])
				})

				Convey("append_message is false", func() {
					sendersNameToType := make(map[string]string)
					settings["append_message"] = false

					err := sender.Init(settings, logger, location, "15:04", sendersNameToType)
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
					So(sendersNameToType["twilio voice"], ShouldEqual, settings["type"])
				})
			})
		})
	})
}
