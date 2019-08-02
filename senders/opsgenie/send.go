package opsgenie

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
)

// SendEvents sends the events list and message to opsgenie
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	_, err := sender.client.Create(nil, &alert.CreateAlertRequest{
		Message:     "message1",
		Alias:       "alias1",
		Description: "alert description1",
		Responders: []alert.Responder{
			{Type: alert.EscalationResponder, Name: "TeamA_escalation"},
			{Type: alert.ScheduleResponder, Name: "TeamB_schedule"},
		},
		VisibleTo: []alert.Responder{
			{Type: alert.UserResponder, Username: "testuser@gmail.com"},
			{Type: alert.TeamResponder, Name: "admin"},
		},
		Actions: []string{"action1", "action2"},
		Tags:    []string{"tag1", "tag2"},
		Details: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Entity:   "entity2",
		Source:   "source2",
		Priority: alert.P1,
		User:     "testuser@gmail.com",
		Note:     "alert note2",
	})
	if err != nil {
		return fmt.Errorf("error while creating alert: %s", err)
	}
	return nil
}
