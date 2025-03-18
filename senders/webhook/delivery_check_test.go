package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/moira-alert/moira"
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
			counter := moira.DeliveryTypesCounter{}

			newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
			So(newCheckData, ShouldResemble, deliveryCheckData{})
			So(reschedule, ShouldBeFalse)
			So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryOK: 1})
		})

		for _, newState := range []string{moira.DeliveryStatePending, moira.DeliveryStateException} {
			Convey(fmt.Sprintf("newState = %s", newState), func() {
				Convey("have attempts left", func() {
					checkData := deliveryCheckData{
						AttemptsCount: 1,
					}
					counter := moira.DeliveryTypesCounter{}

					newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
					So(newCheckData, ShouldResemble, checkData)
					So(reschedule, ShouldBeTrue)
					So(counter, ShouldResemble, counter)
				})

				Convey("no attempts left", func() {
					checkData := deliveryCheckData{
						AttemptsCount: maxAttemptsCount,
					}
					counter := moira.DeliveryTypesCounter{}

					newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
					So(newCheckData, ShouldResemble, deliveryCheckData{})
					So(reschedule, ShouldBeFalse)
					So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryChecksStopped: 1})
				})
			})
		}

		Convey("newState = FAILED", func() {
			checkData := deliveryCheckData{
				AttemptsCount: 1,
			}
			newState := moira.DeliveryStateFailed
			counter := moira.DeliveryTypesCounter{}

			newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
			So(newCheckData, ShouldResemble, checkData)
			So(reschedule, ShouldBeFalse)
			So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryFailed: 1})
		})

		for _, newState := range []string{moira.DeliveryStateUserException, "unknownState"} {
			Convey(fmt.Sprintf("newState = %s", newState), func() {
				checkData := deliveryCheckData{
					AttemptsCount: 1,
				}
				counter := moira.DeliveryTypesCounter{}

				newCheckData, reschedule := handleStateTransition(checkData, newState, maxAttemptsCount, &counter)
				So(newCheckData, ShouldResemble, deliveryCheckData{})
				So(reschedule, ShouldBeFalse)
				So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryChecksStopped: 1})
			})
		}
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

func TestSender_CheckNotificationsDelivery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deliveryCheckCfg := getDefaultDeliveryCheckConfig()

	availableResponses := []string{
		`{"important_field":"?"}`,
		`{"important_field":"2"}`,
	}

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
		log:                 logger,
		deliveryCheckConfig: deliveryCheckCfg,
		client:              ts.Client(),
	}

	Convey("Test CheckNotificationsDelivery", t, func() {
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

			marshaledFetchedChecks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedChecks = append(marshaledFetchedChecks, string(marshaled))
			}

			expectedCheckDataToAdd := deliveryCheckData{
				URL:           expectedFetchedDeliveryChecks[1].URL,
				Contact:       expectedFetchedDeliveryChecks[1].Contact,
				TriggerID:     expectedFetchedDeliveryChecks[1].TriggerID,
				AttemptsCount: expectedFetchedDeliveryChecks[1].AttemptsCount + 1,
			}

			marshaledToStore, err := json.Marshal(expectedCheckDataToAdd)
			So(err, ShouldBeNil)

			gotScheduleAgainChecks, counter := sender.CheckNotificationsDelivery(marshaledFetchedChecks)
			So(gotScheduleAgainChecks, ShouldResemble, []string{string(marshaledToStore)})
			So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryOK: 1})
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

			marshaledFetchedChecks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedChecks = append(marshaledFetchedChecks, string(marshaled))
			}

			expectedCheckDataToAdd := deliveryCheckData{
				URL:           expectedFetchedDeliveryChecks[1].URL,
				Contact:       expectedFetchedDeliveryChecks[1].Contact,
				TriggerID:     expectedFetchedDeliveryChecks[1].TriggerID,
				AttemptsCount: expectedFetchedDeliveryChecks[1].AttemptsCount + 1,
			}

			marshaledToStore, err := json.Marshal(expectedCheckDataToAdd)
			So(err, ShouldBeNil)

			gotScheduleAgainChecks, counter := sender.CheckNotificationsDelivery(marshaledFetchedChecks)
			So(gotScheduleAgainChecks, ShouldResemble, []string{string(marshaledToStore)})
			So(counter, ShouldResemble, moira.DeliveryTypesCounter{})
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

			marshaledFetchedChecks := make([]string, 0, len(expectedFetchedDeliveryChecks))
			for _, checkData := range expectedFetchedDeliveryChecks {
				marshaled, err := json.Marshal(checkData)
				So(err, ShouldBeNil)
				marshaledFetchedChecks = append(marshaledFetchedChecks, string(marshaled))
			}

			gotScheduleAgainChecks, counter := sender.CheckNotificationsDelivery(marshaledFetchedChecks)
			So(gotScheduleAgainChecks, ShouldResemble, []string{})
			So(counter, ShouldResemble, moira.DeliveryTypesCounter{DeliveryOK: 1, DeliveryChecksStopped: 1})
		})
	})
}
