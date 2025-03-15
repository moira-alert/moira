package delivery

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_delivery "github.com/moira-alert/moira/mock/notifier/delivery"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func Test_storingDeliveryChecks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mock_moira_alert.NewMockDeliveryCheckerDatabase(mockCtrl)

	const testContactType = "test_contact_type"
	errFromDB := errors.New("some db error")

	controller := NewChecksController(mockDB, nil, testContactType)

	Convey("Test storing delivery checks with controller", t, func() {
		Convey("AddDeliveryChecksData", func() {
			Convey("no error", func() {
				checkToAdd := "check data"
				givenTimestamp := time.Now().Unix()

				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checkToAdd).Return(nil).Times(1)

				err := controller.AddDeliveryChecksData(givenTimestamp, checkToAdd)
				So(err, ShouldBeNil)
			})

			Convey("error from db", func() {
				checkToAdd := "check data"
				givenTimestamp := time.Now().Unix()

				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checkToAdd).Return(errFromDB).Times(1)

				err := controller.AddDeliveryChecksData(givenTimestamp, checkToAdd)
				So(err, ShouldResemble, errFromDB)
			})
		})

		Convey("addManyDeliveryChecksData", func() {
			Convey("no errors", func() {
				checksToAdd := []string{
					"check data 1",
					"check data 2",
					"check data 3",
				}

				givenTimestamp := time.Now().Unix()

				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checksToAdd[0]).Return(nil).Times(1)
				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checksToAdd[1]).Return(nil).Times(1)
				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checksToAdd[2]).Return(nil).Times(1)

				err := controller.addManyDeliveryChecksData(givenTimestamp, checksToAdd)
				So(err, ShouldBeNil)
			})

			Convey("with error form some", func() {
				checksToAdd := []string{
					"check data 1",
					"check data 2",
					"check data 3",
				}

				givenTimestamp := time.Now().Unix()

				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checksToAdd[0]).Return(nil).Times(1)
				mockDB.EXPECT().AddDeliveryChecksData(testContactType, givenTimestamp, checksToAdd[1]).Return(errFromDB).Times(1)

				err := controller.addManyDeliveryChecksData(givenTimestamp, checksToAdd)
				So(err, ShouldResemble, fmt.Errorf("failed to store check data: %w", errFromDB))
			})
		})

		Convey("getDeliveryChecksData", func() {
			Convey("no errors", func() {
				expectedChecks := []string{
					"check data 1",
					"check data 2",
				}
				givenFrom := "-inf"
				givenTo := "+inf"

				mockDB.EXPECT().GetDeliveryChecksData(testContactType, givenFrom, givenTo).Return(expectedChecks, nil).Times(1)

				gotChecks, err := controller.getDeliveryChecksData(givenFrom, givenTo)
				So(err, ShouldBeNil)
				So(gotChecks, ShouldResemble, expectedChecks)
			})

			Convey("with database.ErrNil from db", func() {
				givenFrom := "-inf"
				givenTo := "+inf"

				mockDB.EXPECT().GetDeliveryChecksData(testContactType, givenFrom, givenTo).Return(nil, database.ErrNil).Times(1)

				gotChecks, err := controller.getDeliveryChecksData(givenFrom, givenTo)
				So(err, ShouldBeNil)
				So(gotChecks, ShouldBeNil)
			})

			Convey("with other error from db", func() {
				givenFrom := "-inf"
				givenTo := "+inf"

				mockDB.EXPECT().GetDeliveryChecksData(testContactType, givenFrom, givenTo).Return(nil, errFromDB).Times(1)

				gotChecks, err := controller.getDeliveryChecksData(givenFrom, givenTo)
				So(err, ShouldResemble, errFromDB)
				So(gotChecks, ShouldBeNil)
			})
		})

		Convey("removeDeliveryChecksData", func() {
			Convey("no errors", func() {
				givenFrom := "-inf"
				givenTo := "+inf"

				expectedCount := int64(5)

				mockDB.EXPECT().RemoveDeliveryChecksData(testContactType, givenFrom, givenTo).Return(expectedCount, nil).Times(1)

				err := controller.removeDeliveryChecksData(givenFrom, givenTo)
				So(err, ShouldBeNil)
			})

			Convey("with error from db", func() {
				givenFrom := "-inf"
				givenTo := "+inf"

				mockDB.EXPECT().RemoveDeliveryChecksData(testContactType, givenFrom, givenTo).Return(int64(0), errFromDB).Times(1)

				err := controller.removeDeliveryChecksData(givenFrom, givenTo)
				So(err, ShouldResemble, errFromDB)
			})
		})
	})
}

func TestChecksController_RunDeliveryChecksWorker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mock_moira_alert.NewMockDeliveryCheckerDatabase(mockCtrl)
	mockLock := mock_moira_alert.NewMockLock(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	logger, _ := logging.GetLogger("test delivery")
	mockCheckAction := mock_delivery.NewMockCheckAction(mockCtrl)

	const (
		testContactType   = "test_contact_type"
		checkTimeout      = time.Second
		reschedulingDelay = uint64(10)
	)

	controller := NewChecksController(mockDB, mockLock, testContactType)
	controller.clock = mockClock

	Convey("Test RunDeliveryChecksWorker", t, func() {
		Convey("with closed channel", func() {
			stopChannel := make(chan struct{})
			close(stopChannel)

			lost := make(chan struct{})
			defer close(lost)

			mockLock.EXPECT().Acquire(stopChannel).Return(lost, nil).Times(1)
			mockLock.EXPECT().Release().Times(1)

			controller.RunDeliveryChecksWorker(
				stopChannel,
				logger,
				checkTimeout,
				reschedulingDelay,
				nil,
				mockCheckAction,
			)

			time.Sleep(checkTimeout + 100*time.Millisecond)
		})

		Convey("with not closed channel", func() {
			stopChannel := make(chan struct{})
			lost := make(chan struct{})
			defer close(lost)

			mockLock.EXPECT().Acquire(stopChannel).Return(lost, nil).Times(1)
			mockLock.EXPECT().Release().Times(1)

			fetchTimestamp := int64(1234567)

			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(testContactType, "-inf", strconv.FormatInt(fetchTimestamp, 10)).
				Return(nil, nil)

			controller.RunDeliveryChecksWorker(
				stopChannel,
				logger,
				checkTimeout,
				reschedulingDelay,
				nil,
				mockCheckAction,
			)

			time.Sleep(checkTimeout + 100*time.Millisecond)
			close(stopChannel)
			time.Sleep(time.Millisecond)
		})
	})
}
