package webhook

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Webhook configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	URL      string `mapstructure:"url"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Timeout  int    `mapstructure:"timeout"`
}

// Sender implements moira sender interface via webhook
type Sender struct {
	webhookClients map[string]*webhookClient
	logger         moira.Logger
}

// webhookClient stores data for the webhook client
type webhookClient struct {
	client   *http.Client
	url      string
	user     string
	password string
	headers  map[string]string
	logger   moira.Logger
}

// Init read yaml config
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to webhook config: %w", err)
	}

	if cfg.Name == "" {
		return fmt.Errorf("required name for sender type webhook")
	}

	if cfg.URL == "" {
		return fmt.Errorf("can not read url from config")
	}

	var timeout int
	if cfg.Timeout != 0 {
		timeout = cfg.Timeout
	} else {
		timeout = 30
	}

	client := &webhookClient{
		url:      cfg.URL,
		user:     cfg.User,
		password: cfg.Password,
		headers: map[string]string{
			"User-Agent":   "Moira",
			"Content-Type": "application/json",
		},
		client: &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: &http.Transport{DisableKeepAlives: true},
		},
		logger: opts.Logger,
	}

	if sender.webhookClients == nil {
		sender.webhookClients = make(map[string]*webhookClient)
	}

	sender.webhookClients[cfg.Name] = client
	sender.logger = opts.Logger

	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	webhookClient, ok := sender.webhookClients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	request, err := webhookClient.buildRequest(events, contact, trigger, plots, throttled)
	if request != nil {
		defer request.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}

	response, err := webhookClient.client.Do(request)
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
