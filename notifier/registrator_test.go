package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/metrics"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
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

func Test_senderMetricsMarker(t *testing.T) {
	Convey("Test senderMetricsMarker", t, func() {
		mockController := gomock.NewController(t)
		defer mockController.Finish()

		sendersDeliveryOK := mock_metrics.NewMockMetersCollection(mockController)
		sendersDeliveryFailed := mock_metrics.NewMockMetersCollection(mockController)

		sendersDeliveryOKMeter := mock_metrics.NewMockMeter(mockController)
		sendersDeliveryFailedMeter := mock_metrics.NewMockMeter(mockController)

		notifierMetrics := &metrics.NotifierMetrics{
			SendersDeliveryOK:     sendersDeliveryOK,
			SendersDeliveryFailed: sendersDeliveryFailed,
		}

		const testSenderContactType = "test_contact_type"
		markCount := int64(1)

		sendersDeliveryFailed.EXPECT().
			RegisterMeter(testSenderContactType, getGraphiteSenderIdent(testSenderContactType), "delivery_failed").
			Times(1)
		sendersDeliveryOK.EXPECT().
			RegisterMeter(testSenderContactType, getGraphiteSenderIdent(testSenderContactType), "delivery_ok").
			Times(1)

		metricsMarker := newSenderMetricsMarker(notifierMetrics, testSenderContactType)

		sendersDeliveryFailed.EXPECT().
			GetRegisteredMeter(testSenderContactType).
			Return(sendersDeliveryFailedMeter, true).
			Times(1)
		sendersDeliveryFailedMeter.EXPECT().Mark(markCount).Times(1)

		metricsMarker.MarkDeliveryFailed()

		sendersDeliveryOK.EXPECT().
			GetRegisteredMeter(testSenderContactType).
			Return(sendersDeliveryOKMeter, true).
			Times(1)
		sendersDeliveryOKMeter.EXPECT().Mark(markCount).Times(1)

		metricsMarker.MarkDeliveryOK()

		sendersDeliveryFailed.EXPECT().GetRegisteredMeter(testSenderContactType).Return(nil, false).Times(1)

		metricsMarker.MarkDeliveryFailed()

		sendersDeliveryOK.EXPECT().GetRegisteredMeter(testSenderContactType).Return(nil, false).Times(1)

		metricsMarker.MarkDeliveryOK()
	})
}
