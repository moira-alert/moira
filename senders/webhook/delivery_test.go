package webhook

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func Test_handleStateTransition(t *testing.T) {
	Convey("Test handle state transition", t, func() {
		const maxAttemptsCount = 5

		Convey("newState = OK", func() {
			checkData := deliveryCheckData{
				AttemptsCount: 1,
			}
			newState := moira.DeliveryStateOK
			counter := deliveryTypesCounter{}

			newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
			So(newCheckData, ShouldResemble, deliveryCheckData{})
			So(reschedule, ShouldBeFalse)
			So(counter, ShouldResemble, deliveryTypesCounter{deliveryOK: 1})
		})

		for _, newState := range []string{moira.DeliveryStatePending, moira.DeliveryStateException} {
			Convey(fmt.Sprintf("newState = %s", newState), func() {
				Convey("have attempts left", func() {
					checkData := deliveryCheckData{
						AttemptsCount: 1,
					}
					counter := deliveryTypesCounter{}

					newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
					So(newCheckData, ShouldResemble, checkData)
					So(reschedule, ShouldBeTrue)
					So(counter, ShouldResemble, deliveryTypesCounter{})
				})

				Convey("no attempts left", func() {
					checkData := deliveryCheckData{
						AttemptsCount: maxAttemptsCount,
					}
					counter := deliveryTypesCounter{}

					newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
					So(newCheckData, ShouldResemble, deliveryCheckData{})
					So(reschedule, ShouldBeFalse)
					So(counter, ShouldResemble, deliveryTypesCounter{deliveryStopped: 1})
				})
			})
		}

		Convey("newState = FAILED", func() {
			checkData := deliveryCheckData{
				AttemptsCount: 1,
			}
			newState := moira.DeliveryStateFailed
			counter := deliveryTypesCounter{}

			newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
			So(newCheckData, ShouldResemble, checkData)
			So(reschedule, ShouldBeFalse)
			So(counter, ShouldResemble, deliveryTypesCounter{deliveryFailed: 1})
		})

		for _, newState := range []string{moira.DeliveryStateUserException, "unknownState"} {
			Convey(fmt.Sprintf("newState = %s", newState), func() {
				checkData := deliveryCheckData{
					AttemptsCount: 1,
				}
				counter := deliveryTypesCounter{}

				newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
				So(newCheckData, ShouldResemble, deliveryCheckData{})
				So(reschedule, ShouldBeFalse)
				So(counter, ShouldResemble, deliveryTypesCounter{deliveryStopped: 1})
			})
		}
	})
}

func Test_markMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Test mark metrics", t, func() {
		deliveryOKMeter := mock_metrics.NewMockMeter(mockCtrl)
		deliveryFailedMeter := mock_metrics.NewMockMeter(mockCtrl)
		deliverStoppedMeter := mock_metrics.NewMockMeter(mockCtrl)

		senderMetrics := &metrics.SenderMetrics{
			ContactDeliveryNotificationOK:           deliveryOKMeter,
			ContactDeliveryNotificationFailed:       deliveryFailedMeter,
			ContactDeliveryNotificationCheckStopped: deliverStoppedMeter,
		}

		counter := &deliveryTypesCounter{
			deliveryOK:      1,
			deliveryFailed:  2,
			deliveryStopped: 3,
		}

		deliveryOKMeter.EXPECT().Mark(counter.deliveryOK).Times(1)
		deliveryFailedMeter.EXPECT().Mark(counter.deliveryFailed).Times(1)
		deliverStoppedMeter.EXPECT().Mark(counter.deliveryStopped).Times(1)

		markMetrics(senderMetrics, counter)
	})
}
