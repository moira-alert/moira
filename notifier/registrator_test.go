package notifier

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRegisterSender(t *testing.T) {
	configureNotifier(t, defaultConfig)
	defer afterTest()

	Convey("Test RegisterSender", t, func() {
		Convey("Sender without type", func() {
			senderSettings := map[string]interface{}{}

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldResemble, ErrMissingSenderType)
		})

		Convey("Sender without contact type", func() {
			senderSettings := map[string]interface{}{
				"sender_type": "test_type",
			}

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldResemble, ErrMissingContactType)
		})

		Convey("Senders with equal contact type", func() {
			senderSettings := map[string]interface{}{
				"sender_type":  "test_type",
				"contact_type": "test_contact",
			}

			sender.EXPECT().Init(
				senderSettings,
				standardNotifier.logger,
				standardNotifier.config.Location,
				standardNotifier.config.DateTimeFormat,
			).Return(nil).Times(1)

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldBeNil)

			err = standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldResemble, fmt.Errorf("failed to initialize sender [%s], err [%w]", senderSettings["contact_type"], ErrSenderRegistered))
		})

		Convey("Successfully register sender", func() {
			senderSettings := map[string]interface{}{
				"sender_type":  "test_type",
				"contact_type": "test_contact_new",
			}

			sender.EXPECT().Init(senderSettings, standardNotifier.logger, standardNotifier.config.Location, standardNotifier.config.DateTimeFormat)

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldBeNil)
		})
	})
}

func TestRegisterSendersWithUnknownType(t *testing.T) {
	config := Config{
		SendingTimeout:   time.Millisecond * 10,
		ResendingTimeout: time.Hour * 24,
		Location:         location,
		DateTimeFormat:   dateTimeFormat,
		Senders: []map[string]interface{}{
			{
				"sender_type": "some_type",
			},
		},
	}

	configureNotifier(t, config)
	defer afterTest()

	Convey("Try to register sender with unknown type", t, func() {
		err := standardNotifier.RegisterSenders(dataBase)
		So(err, ShouldResemble, fmt.Errorf("unknown sender type [%s]", config.Senders[0]["sender_type"]))
	})
}

func TestRegisterSendersWithValidType(t *testing.T) {
	config := Config{
		SendingTimeout:   time.Millisecond * 10,
		ResendingTimeout: time.Hour * 24,
		Location:         location,
		DateTimeFormat:   dateTimeFormat,
		Senders: []map[string]interface{}{
			{
				"sender_type":  "slack",
				"contact_type": "slack",
				"api_token":    "123",
				"use_emoji":    true,
			},
		},
	}

	configureNotifier(t, config)
	defer afterTest()

	Convey("Register sender with valid type", t, func() {
		err := standardNotifier.RegisterSenders(dataBase)
		So(err, ShouldBeNil)
	})
}
