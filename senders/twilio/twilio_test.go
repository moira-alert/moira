package twilio

import (
	"fmt"
	"testing"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
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

		opts := moira.InitOptions{
			SenderSettings: settings,
			Logger:         logger,
			Location:       location,
		}

		Convey("no api asid", func() {
			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_sid param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", func() {
			opts.SenderSettings = settings
			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_authtoken param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", func() {
			opts.SenderSettings = settings
			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_fromphone param from config", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", func() {
			opts.SenderSettings = settings
			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", twilioType))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("config sms", func() {
			settings["type"] = "twilio sms"
			opts.SenderSettings = settings
			err := sender.Init(opts)
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
				opts.SenderSettings = settings
				err := sender.Init(opts)
				So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				So(sender, ShouldResemble, Sender{})
			})

			Convey("has voice url", func() {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", func() {
					settings["append_message"] = true
					opts.SenderSettings = settings
					err := sender.Init(opts)
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
					opts.SenderSettings = settings
					err := sender.Init(opts)
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
