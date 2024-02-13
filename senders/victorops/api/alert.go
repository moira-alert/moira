package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateAlertRequest the API request to be made to
// create a victorops alert
type CreateAlertRequest struct {
	MessageType       MessageType `json:"message_type,omitempty"`
	EntityID          string      `json:"entity_id,omitempty"`
	EntityDisplayName string      `json:"entity_display_name,omitempty"`
	StateMessage      string      `json:"state_message,omitempty"`
	StateStartTime    int64       `json:"state_start_time,omitempty"`
	TriggerURL        string      `json:"trigger_url,omitempty"`
	ImageURL          string      `json:"image_url,omitempty"`
	Timestamp         int64       `json:"timestamp,omitempty"`
	MonitoringTool    string      `json:"monitoring_tool,omitempty"`
}

// MessageType is the type of a victorops alert
type MessageType string

// Various possible MessageTypes
const (
	Critical        MessageType = "CRITICAL"
	Warning         MessageType = "WARNING"
	Acknowledgement MessageType = "ACKNOWLEDGEMENT"
	Info            MessageType = "INFO"
	Recovery        MessageType = "RECOVERY"
)

// CreateAlert creates a new alert in the victorops timeline
func (client *Client) CreateAlert(routingKey string, alert CreateAlertRequest) error {
	if alert.MessageType == "" {
		return fmt.Errorf("field MessageType cannot be empty")
	}

	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("error while encoding json: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("%s/%s", client.routingURL, routingKey), bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error while making the request to victorops: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("victorops API request resulted in error with status %v: %v", resp.StatusCode, string(body))
	}

	return nil
}
