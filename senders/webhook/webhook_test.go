package webhook

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testUser = "testUser"
	testPass = "testPass"

	webhookType = "webhook"
	webhookName = "webhook_name"
)

var logger, _ = logging.GetLogger("webhook")

func TestSender_Init(t *testing.T) {
	Convey("Test Init", t, func() {
		Convey("Init without name", func() {
			senderSettings := map[string]interface{}{
				"type": webhookType,
			}
			sender := Sender{}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
			}

			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("required name for sender type webhook"))
		})

		Convey("Test without url", func() {
			senderSettings := map[string]interface{}{
				"type": webhookType,
				"name": webhookName,
			}
			sender := Sender{}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
			}

			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("can not read url from config"))
		})

		Convey("Init with full config", func() {
			senderSettings := map[string]interface{}{
				"type":     webhookType,
				"name":     webhookName,
				"user":     "user",
				"password": "password",
				"url":      "url",
			}
			sender := Sender{}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)
		})

		Convey("Multiple Init", func() {
			senderSettings1 := map[string]interface{}{
				"type": webhookType,
				"name": webhookName,
				"url":  "url",
			}

			webhookName2 := "webhook_name_2"
			senderSettings2 := map[string]interface{}{
				"type": webhookType,
				"name": webhookName2,
				"url":  "url",
			}

			sender := Sender{}

			opts := moira.InitOptions{
				SenderSettings: senderSettings1,
				Logger:         logger,
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			opts.SenderSettings = senderSettings2

			err = sender.Init(opts)
			So(err, ShouldBeNil)

			So(len(sender.webhookClients), ShouldEqual, 2)
		})
	})
}

func TestSender_SendEvents(t *testing.T) {
	Convey("Receive test webhook", t, func() {
		ts := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					status, err := testRequestURL(r)
					if err != nil {
						w.WriteHeader(status)
						w.Write([]byte(err.Error())) //nolint
					}
					status, err = testRequestHeaders(r)
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
				},
			),
		)
		defer ts.Close()

		senderSettings := map[string]interface{}{
			"name":     webhookName,
			"type":     webhookType,
			"url":      fmt.Sprintf("%s/%s", ts.URL, moira.VariableTriggerID),
			"user":     testUser,
			"password": testPass,
		}

		opts := moira.InitOptions{
			SenderSettings: senderSettings,
			Logger:         logger,
		}

		sender := Sender{}
		err := sender.Init(opts)
		So(err, ShouldBeNil)

		err = sender.SendEvents(testEvents, testContact, testTrigger, testPlot, false)
		So(err, ShouldBeNil)
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

func testRequestHeaders(r *http.Request) (int, error) {
	expectedHeaders := map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
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
	if actualUser != testUser || actualPass != testPass {
		actualCred := fmt.Sprintf("user: %s, pass: %s", actualUser, actualPass)
		expectedCred := fmt.Sprintf("user: %s, pass: %s", testUser, testPass)
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
