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
		Convey("With sender without type", func() {
			senderSettings := map[string]interface{}{}

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldResemble, fmt.Errorf("failed to retrieve sender type from sender settings"))
		})

		Convey("With sender one type", func() {
			senderSettings := map[string]interface{}{
				"type": "test",
			}

			sender.EXPECT().Init(senderSettings, logger, location, dateTimeFormat).Return(nil)

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldBeNil)
		})

		Convey("With multiple senders one type", func() {
			senderSettings := map[string]interface{}{
				"name": "test_name",
				"type": "test",
			}

			Convey("With first sender", func() {
				sender.EXPECT().Init(senderSettings, logger, location, dateTimeFormat).Return(nil)

				err := standardNotifier.RegisterSender(senderSettings, sender)
				So(err, ShouldBeNil)
			})

			senderSettings["name"] = "test_name_2"

			Convey("With second sender", func() {
				sender.EXPECT().Init(senderSettings, logger, location, dateTimeFormat).Return(nil)

				err := standardNotifier.RegisterSender(senderSettings, sender)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestRegisterSendersWithoutType(t *testing.T) {
	config := Config{
		SendingTimeout:   time.Millisecond * 10,
		ResendingTimeout: time.Hour * 24,
		Location:         location,
		DateTimeFormat:   dateTimeFormat,
		Senders: []map[string]interface{}{
			{
				"test": map[string]string{},
			},
		},
	}

	configureNotifier(t, config)
	defer afterTest()

	Convey("With sender without type", t, func() {
		err := standardNotifier.RegisterSenders(dataBase)
		So(err, ShouldResemble, fmt.Errorf("failed to get sender type from settings"))
	})
}
