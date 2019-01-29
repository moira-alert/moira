package webhook

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testUser   = "testUser"
	testPass   = "testPass"
	testHeader = "testHeader"
)

var logger, _ = logging.GetLogger("")

func TestSender_SendEvents(t *testing.T) {
	Convey("Receive test webhook", t, func() {
		ts := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					status, err := testRequest(r)
					w.WriteHeader(status)
					if err != nil {
						w.Write([]byte(err.Error()))
					}
				},
			),
		)
		defer ts.Close()

		senderSettings := map[string]string{
			"name":          "testWebhook",
			"url":           ts.URL,
			"allowed_codes": "200, 201",
			"user":          testUser,
			"password":      testPass,
			"headers":       fmt.Sprintf("{'TestHeader':'%s'}", testHeader),
		}
		sender := Sender{}
		if err := sender.Init(senderSettings, logger, time.UTC, ""); err != nil {
			t.Fatal(err)
		}

		err := sender.SendEvents(testEvents, testContact, testTrigger, testPlot, true)
		So(err, ShouldBeNil)
	})
}

func testRequest(r *http.Request) (int, error) {
	requestBodyBuff := bytes.NewBuffer([]byte{})
	err := r.Write(requestBodyBuff)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	actualJSON, err := getLastLine(requestBodyBuff.String())
	if err != nil {
		return http.StatusInternalServerError, err
	}
	actualJSON, expectedJSON := prepareStrings(actualJSON, expectedPayload, "")
	if actualJSON != expectedJSON {
		return http.StatusBadRequest, fmt.Errorf("invalid json body: %s\nexpected: %s", actualJSON, expectedJSON)
	}
	actualHeader := r.Header.Get("TestHeader")
	if actualHeader != testHeader {
		return http.StatusBadRequest, fmt.Errorf("invalid header value: %s\nexpected: %s", actualHeader, testHeader)
	}
	authHeader := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	payload, err := base64.StdEncoding.DecodeString(authHeader[1])
	if err != nil {
		return http.StatusInternalServerError, err
	}
	pair := strings.SplitN(string(payload), ":", 2)
	actualUser, actualPass := pair[0], pair[1]
	if actualUser != testUser || actualPass != testPass {
		actualCred := fmt.Sprintf("user: %s, pass: %s", actualUser, actualPass)
		expectedCred := fmt.Sprintf("user: %s, pass: %s", testUser, testPass)
		return http.StatusBadRequest, fmt.Errorf("invalid credentials: %s\nexpected: %s", actualCred, expectedCred)
	}
	return http.StatusCreated, nil
}
