package mattermost

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	defaultURL         = "https://mattermost.com/"
	defaultAPIToken    = "test-api-token"
	defaultFrontURI    = "test-front-uri"
	defaultInsecureTLS = true
	defaultUseEmoji    = true
	defaultEmoji       = "test-emoji"
)

var defaultEmojiMap = map[string]string{
	"OK": ":dance_mops:",
}

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	Convey("Init tests", t, func() {
		sender := Sender{}

		validatorErr := validator.ValidationErrors{}

		Convey("With empty url", func() {
			senderSettings := map[string]interface{}{
				"api_token": defaultAPIToken,
				"front_uri": defaultFrontURI,
			}

			err := sender.Init(senderSettings, logger, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With empty api_token", func() {
			senderSettings := map[string]interface{}{
				"url":       defaultURL,
				"front_uri": defaultFrontURI,
			}

			err := sender.Init(senderSettings, logger, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With empty front_uri", func() {
			senderSettings := map[string]interface{}{
				"url":       defaultURL,
				"api_token": defaultAPIToken,
			}

			err := sender.Init(senderSettings, logger, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With full config", func() {
			senderSettings := map[string]interface{}{
				"url":           defaultURL,
				"api_token":     defaultAPIToken,
				"front_uri":     defaultFrontURI,
				"insecure_tls":  defaultInsecureTLS,
				"use_emoji":     defaultUseEmoji,
				"default_emoji": defaultEmoji,
				"emoji_map":     defaultEmojiMap,
			}

			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldBeNil)
		})
	})
}
