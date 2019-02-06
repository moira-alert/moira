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
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testUser   = "testUser"
	testPass   = "testPass"
	testHeader = "testHeader"
)

var testHeaders = map[string]string{testHeader: "testHeaderValue"}

var logger, _ = logging.GetLogger("webhook")

func TestSender_SendEvents(t *testing.T) {
	Convey("Receive test webhook", t, func() {
		ts := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					status, err := testRequestURL(r)
					if err != nil {
						w.WriteHeader(status)
						w.Write([]byte(err.Error()))
					}
					status, err = testRequestHeaders(r)
					if err != nil {
						w.WriteHeader(status)
						w.Write([]byte(err.Error()))
					}
					status, err = testRequestBody(r)
					if err != nil {
						w.Write([]byte(err.Error()))
						w.WriteHeader(status)
					}
					w.WriteHeader(status)
				},
			),
		)
		defer ts.Close()

		senderSettings := map[string]string{
			"name":     "testWebhook",
			"url":      fmt.Sprintf("%s/${trigger_id}", ts.URL),
			"user":     testUser,
			"password": testPass,
			"headers":  fmt.Sprintf("{'%s':'%s'}", testHeader, testHeaders[testHeader]),
		}
		sender := Sender{}
		err := sender.Init(senderSettings, logger, time.UTC, "")
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
	actualHeaderValue, expectedHeaderValue := r.Header.Get(testHeader), testHeaders[testHeader]
	if actualHeaderValue != expectedHeaderValue {
		return http.StatusBadRequest, fmt.Errorf("invalid test header value: %s\nexpected: %s", actualHeaderValue, expectedHeaderValue)
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
