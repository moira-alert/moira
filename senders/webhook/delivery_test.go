package webhook

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestSender_performSingleDeliveryCheck(t *testing.T) {
	const (
		user     = "rmuser"
		password = "rmpassword"
	)

	headers := map[string]string{
		"User-Agent": "Moira",
		"HeaderOne":  "1",
		"HeaderTwo":  "two",
	}

	sender := Sender{
		log: logger,
		deliveryConfig: deliveryCheckConfig{
			User:     user,
			Password: password,
			Headers:  headers,
		},
	}

	availableResponses := []string{
		`{"some_value":"#"}`,
		`{"some_value":"abracadabra"}`,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			writer.Write([]byte(fmt.Sprintf("expected request with GET method, got: %s", request.Method))) //nolint
			return
		}

		status, err := testRequestHeaders(request, headers, user, password)
		if err != nil {
			writer.WriteHeader(status)
			writer.Write([]byte(err.Error())) //nolint
			return
		}

		strIndex := request.URL.Query().Get("bodyIdx")
		index, err := strconv.ParseInt(strIndex, 10, 64)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte(fmt.Sprintf("failed to parse query param 'bodyIdx': %s", err))) //nolint
			return
		}

		if index < 0 || index >= int64(len(availableResponses)) {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte(fmt.Sprintf("index out of range %v of %v", index, len(availableResponses)))) //nolint
			return
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(availableResponses[index])) //nolint
	}))
	defer ts.Close()

	Convey("Test performing single delivery check", t, func() {
		Convey("with not allowed response code", func() {
			sender.client = ts.Client()

			checkData := deliveryCheckData{
				URL:           ts.URL + "/?bodyIdx=a",
				AttemptsCount: 1,
			}

			newCheckData, newState := sender.performSingleDeliveryCheck(checkData)
			So(newState, ShouldResemble, moira.DeliveryStateException)
			So(newCheckData, ShouldResemble, deliveryCheckData{
				URL:           checkData.URL,
				AttemptsCount: checkData.AttemptsCount + 1,
			})
		})

		Convey("with allowed response code, with DeliveryStateOK expected", func() {
			sender.client = ts.Client()
			sender.deliveryConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.some_value "#" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

			checkData := deliveryCheckData{
				URL:           ts.URL + "/?bodyIdx=0",
				AttemptsCount: 1,
			}

			newCheckData, newState := sender.performSingleDeliveryCheck(checkData)
			So(newState, ShouldResemble, moira.DeliveryStateOK)
			So(newCheckData, ShouldResemble, deliveryCheckData{
				URL:           checkData.URL,
				AttemptsCount: checkData.AttemptsCount + 1,
			})
		})

		Convey("with allowed response code, with DeliveryStatePending expected", func() {
			sender.client = ts.Client()
			sender.deliveryConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.some_value "#" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

			checkData := deliveryCheckData{
				URL:           ts.URL + "/?bodyIdx=1",
				AttemptsCount: 2,
			}

			newCheckData, newState := sender.performSingleDeliveryCheck(checkData)
			So(newState, ShouldResemble, moira.DeliveryStatePending)
			So(newCheckData, ShouldResemble, deliveryCheckData{
				URL:           checkData.URL,
				AttemptsCount: checkData.AttemptsCount + 1,
			})
		})

		Convey("with allowed response code but unknown state returned", func() {
			sender.client = ts.Client()
			sender.deliveryConfig.CheckTemplate = `{{ .DeliveryCheckResponse.some_value }}`

			checkData := deliveryCheckData{
				URL:           ts.URL + "/?bodyIdx=1",
				AttemptsCount: 3,
			}

			newCheckData, newState := sender.performSingleDeliveryCheck(checkData)
			So(newState, ShouldResemble, "abracadabra")
			So(newCheckData, ShouldResemble, deliveryCheckData{
				URL:           checkData.URL,
				AttemptsCount: checkData.AttemptsCount + 1,
			})
		})
	})
}
