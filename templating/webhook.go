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
