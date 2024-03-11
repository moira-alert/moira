package templating

type Contact struct {
	Type  string
	Value string
}

type webhookBodyPopulater struct {
	Contact *Contact
}

func NewWebhookBodyPopulater(contact *Contact) *webhookBodyPopulater {
	return &webhookBodyPopulater{
		Contact: contact,
	}
}

func (templateData *webhookBodyPopulater) Populate(tmpl string) (string, error) {
	return populate(tmpl, templateData)
}
