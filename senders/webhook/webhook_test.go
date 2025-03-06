package webhook

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

const (
	testUser = "testUser"
	testPass = "testPass"
	testURL  = "http://localhost:8080"
	testBody = "testBody"

	dateTimeFormat = "15:04 02.01.2006"
)

var (
	logger, _      = logging.GetLogger("webhook")
	location, _    = time.LoadLocation("UTC")
	defaultHeaders = map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}
)

func TestSender_Init(t *testing.T) {
	Convey("Test Init function", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		validatorErr := validator.ValidationErrors{}

		Convey("With empty url", func() {
			settings := map[string]interface{}{}
			sender := Sender{}

			err := sender.Init(settings, logger, location, dateTimeFormat)
			So(errors.As(err, &validatorErr), ShouldBeTrue)
		})

		Convey("With only url", func() {
			settings := map[string]interface{}{
				"url": testURL,
			}
			sender := Sender{}

			err := sender.Init(settings, logger, location, dateTimeFormat)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{
				url:     testURL,
				headers: defaultHeaders,
				client: &http.Client{
					Timeout:   30 * time.Second,
					Transport: &http.Transport{DisableKeepAlives: true},
				},
				log:                 logger,
				deliveryCheckConfig: getDefaultDeliveryCheckConfig(),
				clock:               clock.NewSystemClock(),
			})
		})

		Convey("With full provided data", func() {
			settings := map[string]interface{}{
				"contact_type": testContact.Type,
				"url":          testURL,
				"body":         testBody,
				"user":         testUser,
				"password":     testPass,
				"headers": map[string]string{
					"testHeader": "test",
				},
				"timeout": 120,
			}
			sender := Sender{}
			expectedHeaders := maps.Clone(defaultHeaders)
			expectedHeaders["testHeader"] = "test"

			err := sender.Init(settings, logger, location, dateTimeFormat)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{
				contactType: testContact.Type,
				url:         testURL,
				body:        testBody,
				user:        testUser,
				password:    testPass,
				headers:     expectedHeaders,
				client: &http.Client{
					Timeout:   120 * time.Second,
					Transport: &http.Transport{DisableKeepAlives: true},
				},
				log:                 logger,
				deliveryCheckConfig: getDefaultDeliveryCheckConfig(),
				clock:               clock.NewSystemClock(),
			})
		})

		Convey("With url and sender metrics", func() {
			senderMetrics := &metrics.SenderMetrics{}

			settings := map[string]interface{}{
				"url":            testURL,
				senderMetricsKey: senderMetrics,
			}

			sender := Sender{}
			err := sender.Init(settings, logger, location, dateTimeFormat)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{
				url:     testURL,
				headers: defaultHeaders,
				client: &http.Client{
					Timeout:   30 * time.Second,
					Transport: &http.Transport{DisableKeepAlives: true},
				},
				log:                 logger,
				metrics:             senderMetrics,
				deliveryCheckConfig: getDefaultDeliveryCheckConfig(),
				clock:               clock.NewSystemClock(),
			})
		})

		Convey("With delivery checks enabled", func() {
			Convey("but empty url_template", func() {
				settings := map[string]interface{}{
					"url": testURL,
					"delivery_check": map[string]interface{}{
						"enabled": true,
					},
				}

				sender := Sender{}
				err := sender.Init(settings, logger, location, dateTimeFormat)
				So(errors.As(err, &validatorErr), ShouldBeTrue)
			})

			Convey("but empty check_template", func() {
				settings := map[string]interface{}{
					"url": testURL,
					"delivery_check": map[string]interface{}{
						"enabled":      true,
						"url_template": "https://example.com/",
					},
				}

				sender := Sender{}
				err := sender.Init(settings, logger, location, dateTimeFormat)
				So(errors.As(err, &validatorErr), ShouldBeTrue)
			})

			Convey("but nil sender metrics", func() {
				settings := map[string]interface{}{
					"url": testURL,
					"delivery_check": map[string]interface{}{
						"enabled":        true,
						"url_template":   "https://example.com/",
						"check_template": "{{ if eq .DeliveryCheckResponse.someValues 0 }}.DeliveryStateOK{{ else }}.DeliveryStatePending{{ end }}",
					},
				}

				sender := Sender{}
				err := sender.Init(settings, logger, location, dateTimeFormat)
				So(err, ShouldResemble, errNilMetricsOnDeliveryCheck)
			})
		})
	})
}

func TestSender_SendEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := `{"requestID": "some-request-id"}`

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				status, err := testRequestURL(r)
				if err != nil {
					w.WriteHeader(status)
					w.Write([]byte(err.Error())) //nolint
				}
				status, err = testRequestHeaders(r,
					map[string]string{
						"Content-Type": "application/json",
					},
					testUser,
					testPass)
				if err != nil {
					w.WriteHeader(status)
					w.Write([]byte(err.Error())) //nolint
				}
				status, err = testRequestBody(r)
				if err != nil {
					w.Write([]byte(err.Error())) //nolint
					w.WriteHeader(status)
				}
				w.WriteHeader(status)
				w.Write([]byte(body))
			},
		),
	)
	defer ts.Close()

	Convey("Receive test webhook", t, func() {
		Convey("with delivery check disabled", func() {
			senderSettings := map[string]interface{}{
				"url":      fmt.Sprintf("%s/%s", ts.URL, moira.VariableTriggerID),
				"user":     testUser,
				"password": testPass,
			}
			sender := Sender{}
			err := sender.Init(senderSettings, logger, time.UTC, "")
			So(err, ShouldBeNil)

			err = sender.SendEvents(testEvents, testContact, testTrigger, testPlot, false)
			So(err, ShouldBeNil)
		})

		Convey("with delivery check enabled", func() {
			mockDB := mock_moira_alert.NewMockDeliveryCheckerDatabase(mockCtrl)
			mockClock := mock_clock.NewMockClock(mockCtrl)

			sender := Sender{
				contactType:         testContact.Type,
				url:                 fmt.Sprintf("%s/%s", ts.URL, moira.VariableTriggerID),
				user:                testUser,
				password:            testPass,
				headers:             defaultHeaders,
				log:                 logger,
				metrics:             &metrics.SenderMetrics{},
				Database:            mockDB,
				clock:               mockClock,
				client:              ts.Client(),
				deliveryCheckConfig: getDefaultDeliveryCheckConfig(),
			}
			sender.deliveryCheckConfig.Enabled = true
			sender.deliveryCheckConfig.URLTemplate = "http://example.com/{{ .SendAlertResponse.requestID }}"
			sender.deliveryCheckConfig.CheckTemplate = "{{ .StateConstants.DeliveryCheckOK }}"

			timestamp := int64(12345)
			expectedDeliveryCheckData := deliveryCheckData{
				Timestamp:     timestamp,
				URL:           "http://example.com/some-request-id",
				Contact:       testContact,
				TriggerID:     testTrigger.ID,
				AttemptsCount: 0,
			}

			marshaled, err := json.Marshal(expectedDeliveryCheckData)
			So(err, ShouldBeNil)

			mockClock.EXPECT().NowUnix().Return(timestamp).Times(1)
			mockDB.EXPECT().AddDeliveryChecksData(testContact.Type, timestamp, string(marshaled)).Return(nil).Times(1)

			err = sender.SendEvents(testEvents, testContact, testTrigger, testPlot, false)
			So(err, ShouldBeNil)
		})
	})
}

func testRequestURL(r *http.Request) (int, error) {
	actualPath := r.URL.EscapedPath()
	expectedPath := fmt.Sprintf("/%s", url.PathEscape(testTrigger.ID))
	if actualPath != expectedPath {
		return http.StatusBadRequest, fmt.Errorf("invalid url path: %s\nexpected: %s", actualPath, expectedPath)
	}
	return http.StatusCreated, nil
}

func testRequestHeaders(r *http.Request, customExpectedHeaders map[string]string, expectedUser, expectedPassword string) (int, error) {
	expectedHeaders := map[string]string{
		"User-Agent": "Moira",
	}

	for headerName, headerValue := range customExpectedHeaders {
		expectedHeaders[headerName] = headerValue
	}

	for headerName, headerValue := range expectedHeaders {
		actualHeaderValue := r.Header.Get(headerName)
		if actualHeaderValue != headerValue {
			return http.StatusBadRequest, fmt.Errorf("invalid header value: %s\nexpected: %s", actualHeaderValue, headerValue)
		}
	}
	authHeader := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	authPayload, err := base64.StdEncoding.DecodeString(authHeader[1])
	if err != nil {
		return http.StatusInternalServerError, err
	}
	authPair := strings.SplitN(string(authPayload), ":", 2)
	actualUser, actualPass := authPair[0], authPair[1]
	if actualUser != expectedUser || actualPass != expectedPassword {
		actualCred := fmt.Sprintf("user: %s, pass: %s", actualUser, actualPass)
		expectedCred := fmt.Sprintf("user: %s, pass: %s", expectedUser, expectedPassword)
		return http.StatusBadRequest, fmt.Errorf("invalid credentials: %s\nexpected: %s", actualCred, expectedCred)
	}
	return http.StatusCreated, nil
}

func testRequestBody(r *http.Request) (int, error) {
	requestBodyBuff := bytes.NewBuffer([]byte{})
	err := r.Write(requestBodyBuff)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	actualJSON, err := getLastLine(requestBodyBuff.String())
	if err != nil {
		return http.StatusInternalServerError, err
	}
	actualJSON, expectedJSON := prepareStrings(actualJSON, expectedStateChangePayload)
	if actualJSON != expectedJSON {
		return http.StatusBadRequest, fmt.Errorf("invalid json body: %s\nexpected: %s", actualJSON, expectedJSON)
	}
	return http.StatusCreated, nil
}

func getLastLine(longString string) (string, error) {
	reader := bytes.NewReader([]byte(longString))
	var lastLine string
	s := bufio.NewScanner(reader)
	for s.Scan() {
		lastLine = s.Text()
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return lastLine, nil
}
