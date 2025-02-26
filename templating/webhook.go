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
	Contact      *Contact
	SendResponse map[string]interface{}
}

func NewWebhookDeliveryCheckURLPopulater(contact *Contact, sendRsp map[string]interface{}) *webhookDeliveryCheckURLPopulater {
	return &webhookDeliveryCheckURLPopulater{
		Contact:      contact,
		SendResponse: sendRsp,
	}
}

// Populate populates the given template with contact data.
func (templateData *webhookDeliveryCheckURLPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}

type webhookDeliveryCheckPopulater struct {
	Contact               *Contact
	DeliveryCheckResponse map[string]interface{}
}

func NewWebhookDeliveryCheckPopulater(contact *Contact, sendRsp map[string]interface{}) *webhookDeliveryCheckPopulater {
	return &webhookDeliveryCheckPopulater{
		Contact:               contact,
		DeliveryCheckResponse: sendRsp,
	}
}

// Populate populates the given template with contact data.
func (templateData *webhookDeliveryCheckPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}
