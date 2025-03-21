package webhook

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// Structure that represents the Webhook configuration in the YAML file.
type config struct {
	URL         string            `mapstructure:"url" validate:"required"`
	Body        string            `mapstructure:"body"`
	Headers     map[string]string `mapstructure:"headers"`
	User        string            `mapstructure:"user"`
	Password    string            `mapstructure:"password"`
	Timeout     int               `mapstructure:"timeout"`
	InsecureTLS bool              `mapstructure:"insecure_tls"`
}

// Sender implements moira sender interface via webhook.
type Sender struct {
	url      string
	body     string
	user     string
	password string
	headers  map[string]string
	client   *http.Client
	log      moira.Logger
	metrics  *metrics.SenderMetrics
}

const senderMetricsKey = "sender_metrics"

// Init read yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to webhook config: %w", err)
	}

	if err = moira.ValidateStruct(cfg); err != nil {
		return fmt.Errorf("webhook config validation error: %w", err)
	}

	sender.url = cfg.URL
	sender.body = cfg.Body
	sender.user, sender.password = cfg.User, cfg.Password

	sender.headers = map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}

	for header, value := range cfg.Headers {
		sender.headers[header] = value
	}

	var timeout int
	if cfg.Timeout != 0 {
		timeout = cfg.Timeout
	} else {
		timeout = 30
	}

	sender.log = logger
	sender.client = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureTLS,
			},
		},
	}

	senderSettingsMap := senderSettings.(map[string]interface{})
	if val, ok := senderSettingsMap[senderMetricsKey]; ok {
		sender.metrics = val.(*metrics.SenderMetrics)
	}

	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	request, err := sender.buildRequest(events, contact, trigger, plots, throttled)
	if request != nil {
		defer request.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	response, err := sender.client.Do(request)
	if response != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
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
