package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
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
		deliveryCheckConfig: deliveryCheckConfig{
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
			sender.deliveryCheckConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.some_value "#" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

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
			sender.deliveryCheckConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.some_value "#" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

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
			sender.deliveryCheckConfig.CheckTemplate = `{{ .DeliveryCheckResponse.some_value }}`

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

func Test_prepareDeliveryCheck(t *testing.T) {
	Convey("Test prepareDeliveryCheck", t, func() {
		Convey("with error after populating url template", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id "

			actualCheckData, err := prepareDeliveryCheck(testContact, nil, urlTemplate, testTrigger.ID)
			So(err, ShouldNotBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with empty rsp map", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(testContact, map[string]interface{}{}, urlTemplate, testTrigger.ID)
			So(err, ShouldBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{
				URL:           "https://example.com/",
				Contact:       testContact,
				TriggerID:     testTrigger.ID,
				AttemptsCount: 0,
			})
		})

		Convey("with non empty rsp map", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{
				URL:           "https://example.com/123456",
				Contact:       testContact,
				TriggerID:     testTrigger.ID,
				AttemptsCount: 0,
			})
		})

		Convey("with not url format", func() {
			urlTemplate := "example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldNotBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with bad scheme", func() {
			urlTemplate := "smtp://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldResemble, fmt.Errorf("got bad url for check request: %w, url: smtp://example.com/123456", fmt.Errorf("bad url scheme: smtp")))
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with empty host", func() {
			urlTemplate := "https:///{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldResemble, fmt.Errorf("got bad url for check request: %w, url: https:///123456", fmt.Errorf("host is empty")))
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})
	})
}

func TestSender_checkNotificationsDelivery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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
	deliveryCheckCfg := getDefaultDeliveryCheckConfig()

	availableResponses := []string{
		`{"important_field":"?"}`,
		`{"important_field":"2"}`,
	}

	fetchTimestamp := int64(123456)
	fetchTimestampStr := strconv.FormatInt(fetchTimestamp, 10)

	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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

	sender := Sender{
		contactType:         testContact.Type,
		log:                 logger,
		metrics:             senderMetrics,
		Database:            mockDB,
		deliveryCheckConfig: deliveryCheckCfg,
		clock:               mockClock,
		client:              ts.Client(),
	}

	Convey("Test checkNotificationsDelivery", t, func() {
		Convey("when database.ErrNil is returned on fetching checks data", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(nil, database.ErrNil).Times(1)

			err := sender.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("when no checks data while fetching", func() {
			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return([]string{}, nil).Times(1)

			err := sender.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("with unique checks data", func() {
			sender.deliveryCheckConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.important_field "?" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

			expectedFetchedDeliveryChecks := []deliveryCheckData{
				{
					URL:           ts.URL + "/?bodyIdx=0",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 0,
				},
				{
					URL:           ts.URL + "/?bodyIdx=1",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 1,
				},
			}

			marshaledFetchedCheks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedCheks = append(marshaledFetchedCheks, string(marshaled))
			}

			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(marshaledFetchedCheks, nil).Times(1)

			timestamp := int64(123457)
			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)
			storeTimestamp := timestamp + int64(sender.deliveryCheckConfig.ReschedulingDelay)

			expectedCheckDataToAdd := deliveryCheckData{
				Timestamp:     storeTimestamp,
				URL:           expectedFetchedDeliveryChecks[1].URL,
				Contact:       expectedFetchedDeliveryChecks[1].Contact,
				TriggerID:     expectedFetchedDeliveryChecks[1].TriggerID,
				AttemptsCount: expectedFetchedDeliveryChecks[1].AttemptsCount + 1,
			}

			marshaledToStore, err := json.Marshal(expectedCheckDataToAdd)
			So(err, ShouldBeNil)

			mockDB.EXPECT().AddDeliveryChecksData(sender.contactType, storeTimestamp, string(marshaledToStore)).Return(nil).Times(1)
			mockDB.EXPECT().RemoveDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(int64(len(expectedFetchedDeliveryChecks)), nil).Times(1)

			deliveryOKMetricsMock.EXPECT().Mark(int64(1)).Times(1)
			deliveryFailedMetricsMock.EXPECT().Mark(int64(0)).Times(1)
			deliveryCheckStoppedMetricsMock.EXPECT().Mark(int64(0)).Times(1)

			err = sender.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("with duplicated check data", func() {
			sender.deliveryCheckConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.important_field "?" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

			expectedFetchedDeliveryChecks := []deliveryCheckData{
				{
					URL:           ts.URL + "/?bodyIdx=1",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 0,
				},
				{
					URL:           ts.URL + "/?bodyIdx=1",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 1,
				},
			}

			marshaledFetchedCheks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedCheks = append(marshaledFetchedCheks, string(marshaled))
			}

			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(marshaledFetchedCheks, nil).Times(1)

			timestamp := int64(123457)
			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)
			storeTimestamp := timestamp + int64(sender.deliveryCheckConfig.ReschedulingDelay)

			expectedCheckDataToAdd := deliveryCheckData{
				Timestamp:     storeTimestamp,
				URL:           expectedFetchedDeliveryChecks[1].URL,
				Contact:       expectedFetchedDeliveryChecks[1].Contact,
				TriggerID:     expectedFetchedDeliveryChecks[1].TriggerID,
				AttemptsCount: expectedFetchedDeliveryChecks[1].AttemptsCount + 1,
			}

			marshaledToStore, err := json.Marshal(expectedCheckDataToAdd)
			So(err, ShouldBeNil)

			mockDB.EXPECT().AddDeliveryChecksData(sender.contactType, storeTimestamp, string(marshaledToStore)).Return(nil).Times(1)
			mockDB.EXPECT().RemoveDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(int64(len(expectedFetchedDeliveryChecks)), nil).Times(1)

			deliveryOKMetricsMock.EXPECT().Mark(int64(0)).Times(1)
			deliveryFailedMetricsMock.EXPECT().Mark(int64(0)).Times(1)
			deliveryCheckStoppedMetricsMock.EXPECT().Mark(int64(0)).Times(1)

			err = sender.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})

		Convey("with no checks to check again", func() {
			sender.deliveryCheckConfig.CheckTemplate = `{{ if eq .DeliveryCheckResponse.important_field "?" }}{{ .StateConstants.DeliveryStateOK }}{{ else }}{{ .StateConstants.DeliveryStatePending }}{{ end }}`

			expectedFetchedDeliveryChecks := []deliveryCheckData{
				{
					URL:           ts.URL + "/?bodyIdx=0",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 0,
				},
				{
					URL:           ts.URL + "/?bodyIdx=1",
					Contact:       testContact,
					TriggerID:     testTrigger.ID,
					AttemptsCount: 4,
				},
			}

			marshaledFetchedCheks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedCheks = append(marshaledFetchedCheks, string(marshaled))
			}

			mockClock.EXPECT().NowUnix().Return(fetchTimestamp).Times(1)
			mockDB.EXPECT().GetDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(marshaledFetchedCheks, nil).Times(1)

			timestamp := int64(123457)
			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)

			mockDB.EXPECT().RemoveDeliveryChecksData(sender.contactType, "-inf", fetchTimestampStr).Return(int64(len(expectedFetchedDeliveryChecks)), nil).Times(1)

			deliveryOKMetricsMock.EXPECT().Mark(int64(1)).Times(1)
			deliveryFailedMetricsMock.EXPECT().Mark(int64(0)).Times(1)
			deliveryCheckStoppedMetricsMock.EXPECT().Mark(int64(1)).Times(1)

			err := sender.checkNotificationsDelivery()
			So(err, ShouldBeNil)
		})
	})
}
