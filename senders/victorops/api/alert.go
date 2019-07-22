package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		return fmt.Errorf("MessageType field cannot be empty")
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(alert)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", client.routingURL, routingKey), body)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error while making the request to victorops: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("victorops API request resulted in error with status %v: %v", resp.StatusCode, string(body))
	}

	return nil
}
