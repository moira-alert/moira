package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/metrics"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
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

		Convey("Successfully register sender with damaged enable_metrics", func() {
			sendersOkMetrics := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersFailedMetrics := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersDroppedNotifications := mock_metrics.NewMockMetersCollection(mockCtrl)

			notifierMetrics := &metrics.NotifierMetrics{
				ContactsSendingNotificationsOK:     sendersOkMetrics,
				ContactsSendingNotificationsFailed: sendersFailedMetrics,
				ContactsDroppedNotifications:       sendersDroppedNotifications,
			}
			standardNotifier.metrics = notifierMetrics

			senderContactType := "test_contact_new_1"
			senderSettings := map[string]interface{}{
				"sender_type":    "test_type",
				"contact_type":   "test_contact_new_1",
				"enable_metrics": "abracdabra",
			}

			sendersOkMetrics.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_ok").Times(1)
			sendersFailedMetrics.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_failed").Times(1)
			sendersDroppedNotifications.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "notifications_dropped").Times(1)
			sender.EXPECT().Init(senderSettings, standardNotifier.logger, standardNotifier.config.Location, standardNotifier.config.DateTimeFormat)

			err := standardNotifier.RegisterSender(senderSettings, sender)
			So(err, ShouldBeNil)
		})

		Convey("Register sender with additional metrics", func() {
			sendersOkMetrics := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersFailedMetrics := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersDroppedNotifications := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersDeliveryOK := mock_metrics.NewMockMetersCollection(mockCtrl)
			sendersDeliveryFailed := mock_metrics.NewMockMetersCollection(mockCtrl)

			notifierMetrics := &metrics.NotifierMetrics{
				ContactsSendingNotificationsOK:      sendersOkMetrics,
				ContactsSendingNotificationsFailed:  sendersFailedMetrics,
				ContactsDroppedNotifications:        sendersDroppedNotifications,
				ContactsDeliveryNotificationsOK:     sendersDeliveryOK,
				ContactsDeliveryNotificationsFailed: sendersDeliveryFailed,
			}
			standardNotifier.metrics = notifierMetrics

			senderContactType := "test_contact_new_2"
			senderSettings := map[string]interface{}{
				"sender_type":    "test_type",
				"contact_type":   senderContactType,
				"enable_metrics": true,
			}

			sendersOkMetrics.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_ok").Times(1)
			sendersFailedMetrics.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_failed").Times(1)
			sendersDroppedNotifications.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "notifications_dropped").Times(1)
			sendersDeliveryOK.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "delivery_ok").Times(1)
			sendersDeliveryFailed.EXPECT().RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "delivery_failed").Times(1)
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
