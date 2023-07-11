package webhook

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Webhook configuration in the YAML file
type WebHook struct {
	Name          string              `mapstructure:"name"`
	URL           string              `mapstructure:"url"`
	User          string              `mapstructure:"user"`
	Password      string              `mapstructure:"password"`
	CustomHeaders []map[string]string `mapstructure:"custom-headers"`
	Timeout       string              `mapstructure:"timeout,omitempty"`
}

// Sender implements moira sender interface via webhook
type Sender struct {
	url      string
	user     string
	password string
	headers  map[string]string
	client   *http.Client
	log      moira.Logger
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var webhook WebHook
	err := mapstructure.Decode(senderSettings, &webhook)
	if err != nil {
		return fmt.Errorf("decoding error from yaml file to webhook structure: %s", err)
	}

	if webhook.Name == "" {
		return fmt.Errorf("required name for sender type webhook")
	}

	sender.url = webhook.URL
	if sender.url == "" {
		return fmt.Errorf("can not read url from config")
	}

	sender.user, sender.password = webhook.User, webhook.Password

	sender.headers = map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}

	for _, customHeader := range webhook.CustomHeaders {
		sender.headers[customHeader["key"]] = customHeader["value"]
	}

	timeout := 30
	timeoutRaw := webhook.Timeout
	if timeoutRaw != "" {
		timeout, err = strconv.Atoi(timeoutRaw)
		if err != nil {
			return fmt.Errorf("can not read timeout from config: %s", err.Error())
		}
	}

	sender.log = logger
	sender.client = &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	request, err := sender.buildRequest(events, contact, trigger, plots, throttled)
	if request != nil {
		defer request.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}

	response, err := sender.client.Do(request)
	if response != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err.Error())
	}

	if !isAllowedResponseCode(response.StatusCode) {
		var serverResponse string
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			serverResponse = fmt.Sprintf("failed to read response body: %s", err.Error())
		} else {
			serverResponse = string(responseBody)
		}
		return fmt.Errorf("invalid status code: %d, server response: %s", response.StatusCode, serverResponse)
	}

	return nil
}

func isAllowedResponseCode(responseCode int) bool {
	return (responseCode >= http.StatusOK) && (responseCode < http.StatusMultipleChoices)
}
