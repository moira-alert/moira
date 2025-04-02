package templating

// Contact represents a template contact with fields allowed for use in templates.
type Contact struct {
	Type  string
	Value string
}

type webhookBodyPopulater struct {
	Contact *Contact
}

// NewWebhookBodyPopulater creates a new webhook body populater with provided template contact.
func NewWebhookBodyPopulater(contact *Contact) *webhookBodyPopulater {
	return &webhookBodyPopulater{
		Contact: contact,
	}
}

// Populate populates the given template with contact data.
func (templateData *webhookBodyPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}

type webhookDeliveryCheckURLPopulater struct {
	Contact           *Contact
	SendAlertResponse map[string]interface{}
	TriggerID         string
}

// NewWebhookDeliveryCheckURLPopulater creates a new webhook url populater with provided template contact
// and body from response got on send alert request.
func NewWebhookDeliveryCheckURLPopulater(contact *Contact, sendAlertResponse map[string]interface{}, triggerID string) *webhookDeliveryCheckURLPopulater {
	return &webhookDeliveryCheckURLPopulater{
		Contact:           contact,
		SendAlertResponse: sendAlertResponse,
		TriggerID:         triggerID,
	}
}

// Populate populates the given template with contact data and response got on send alert request.
func (templateData *webhookDeliveryCheckURLPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}

type webhookDeliveryCheckPopulater struct {
	Contact               *Contact
	DeliveryCheckResponse map[string]interface{}
	TriggerID             string
}

// NewWebhookDeliveryCheckPopulater creates a new webhook delivery check populater with provided template contact,
// triggerID and body from response got on delivery check request.
func NewWebhookDeliveryCheckPopulater(contact *Contact, deliveryCheckResponse map[string]interface{}, triggerID string) *webhookDeliveryCheckPopulater {
	return &webhookDeliveryCheckPopulater{
		Contact:               contact,
		DeliveryCheckResponse: deliveryCheckResponse,
		TriggerID:             triggerID,
	}
}

// Populate populates the given template with contact data, response got on send alert request and delivery state constants.
func (templateData *webhookDeliveryCheckPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}
