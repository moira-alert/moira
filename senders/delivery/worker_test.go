package delivery

import (
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	mock_delivery "github.com/moira-alert/moira/mock/notifier/delivery"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func Test_markMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deliveryOKMeter := mock_metrics.NewMockMeter(mockCtrl)
	deliveryFailedMeter := mock_metrics.NewMockMeter(mockCtrl)
	deliverStoppedMeter := mock_metrics.NewMockMeter(mockCtrl)

	senderMetrics := &metrics.SenderMetrics{
		ContactDeliveryNotificationOK:           deliveryOKMeter,
		ContactDeliveryNotificationFailed:       deliveryFailedMeter,
		ContactDeliveryNotificationCheckStopped: deliverStoppedMeter,
	}

	counter := &moira.DeliveryTypesCounter{
		DeliveryOK:            1,
		DeliveryFailed:        2,
		DeliveryChecksStopped: 3,
	}

	Convey("Test mark metrics", t, func() {
		Convey("with non nil params", func() {
			deliveryOKMeter.EXPECT().Mark(counter.DeliveryOK).Times(1)
			deliveryFailedMeter.EXPECT().Mark(counter.DeliveryFailed).Times(1)
			deliverStoppedMeter.EXPECT().Mark(counter.DeliveryChecksStopped).Times(1)

			markMetrics(senderMetrics, counter)
		})

		Convey("with nil metrics", func() {
			markMetrics(nil, counter)
		})

		Convey("with nil counter", func() {
			markMetrics(senderMetrics, nil)
		})
	})
}

func TestSender_checkNotificationsDelivery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger, _ := logging.GetLogger("test delivery")

	mockDB := mock_moira_alert.NewMockDeliveryCheckerDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)

	deliveryOKMetricsMock := mock_metrics.NewMockMeter(mockCtrl)
	deliveryFailedMetricsMock := mock_metrics.NewMockMeter(mockCtrl)
	deliveryCheckStoppedMetricsMock := mock_metrics.NewMockMeter(mockCtrl)
	senderMetrics := &metrics.SenderMetrics{
		ContactDeliveryNotificationOK:           deliveryOKMetricsMock,
		ContactDeliveryNotificationFailed:       deliveryFailedMetricsMock,
		ContactDeliveryNotificationCheckStopped: deliveryCheckStoppedMetricsMock,
	}

	fetchTimestamp := int64(123456)
	fetchTimestampStr := strconv.FormatInt(fetchTimestamp, 10)

	reschedulingDelay := uint64(40)
	testContactType := "test_contact_type"

	controller := NewChecksController(mockDB, nil, testContactType)
	mockCheckAction := mock_delivery.NewMockCheckAction(mockCtrl)

	testChecksWorker := newChecksWorker(
		logger,
		mockClock,
		"test_delivery_checks_worker",
		time.Second,
		reschedulingDelay,
		controller,
		senderMetrics,
		mockCheckAction,
	)

	fetchedChecks := []string{
		"check data 1",
		"check data 2",
		"check data 3",
	}

	Convey("Test checkNotificationsDelivery", t, func() {
		Convey("when database.ErrNil is returned on fetching checks data", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return(nil, database.ErrNil).Times(1)

			err := testChecksWorker.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("when no checks data while fetching", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return([]string{}, nil).Times(1)

			err := testChecksWorker.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("with checks data need to check again", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return(fetchedChecks, nil).Times(1)

			timestamp := int64(123457)
			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)
			storeTimestamp := timestamp + int64(reschedulingDelay)

			mockCheckAction.EXPECT().CheckNotificationsDelivery(fetchedChecks).
				Return(
					fetchedChecks[1:],
					moira.DeliveryTypesCounter{
						DeliveryOK: 1,
					}).
				Times(1)

			mockDB.EXPECT().AddDeliveryChecksData(testContactType, storeTimestamp, fetchedChecks[1]).Return(nil).Times(1)
			mockDB.EXPECT().AddDeliveryChecksData(testContactType, storeTimestamp, fetchedChecks[2]).Return(nil).Times(1)
			mockDB.EXPECT().RemoveDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return(int64(len(fetchedChecks)), nil).Times(1)

			deliveryOKMetricsMock.EXPECT().Mark(int64(1)).Times(1)
			deliveryFailedMetricsMock.EXPECT().Mark(int64(0)).Times(1)
			deliveryCheckStoppedMetricsMock.EXPECT().Mark(int64(0)).Times(1)

			err := testChecksWorker.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("with no checks to check again", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return(fetchedChecks, nil).Times(1)

			timestamp := int64(123457)
			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)

			mockCheckAction.EXPECT().CheckNotificationsDelivery(fetchedChecks).
				Return(
					nil,
					moira.DeliveryTypesCounter{
						DeliveryOK:            1,
						DeliveryFailed:        1,
						DeliveryChecksStopped: 1,
					}).
				Times(1)

			mockDB.EXPECT().RemoveDeliveryChecksData(testContactType, "-inf", fetchTimestampStr).Return(int64(len(fetchedChecks)), nil).Times(1)

			deliveryOKMetricsMock.EXPECT().Mark(int64(1)).Times(1)
			deliveryFailedMetricsMock.EXPECT().Mark(int64(1)).Times(1)
			deliveryCheckStoppedMetricsMock.EXPECT().Mark(int64(1)).Times(1)

			err := testChecksWorker.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})
	})
}
