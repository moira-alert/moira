package webhook

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface via Webhook.
// Use NewSender to create instance.
type Sender struct {
	url      string
	user     string
	password string
	headers  map[string]string
	client   *http.Client
	log      moira.Logger
}

// NewSender creates Sender instance.
func NewSender(senderSettings map[string]string, logger moira.Logger) (*Sender, error) {
	sender := &Sender{}

	if senderSettings["name"] == "" {
		return nil, fmt.Errorf("required name for sender type webhook")
	}

	sender.url = senderSettings["url"]
	if sender.url == "" {
		return nil, fmt.Errorf("can not read url from config")
	}

	sender.user, sender.password = senderSettings["user"], senderSettings["password"]

	sender.headers = map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}

	timeout := 30
	if timeoutRaw, ok := senderSettings["timeout"]; ok {
		var err error
		timeout, err = strconv.Atoi(timeoutRaw)
		if err != nil {
			return nil, fmt.Errorf("can not read timeout from config: %s", err.Error())
		}
	}

	sender.log = logger
	sender.client = &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	return sender, nil
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
