package templating

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_TemplateWebhookBody(t *testing.T) {
	Convey("Test webhook body populater", t, func() {
		template := "" +
			"Contact Type: {{ .Contact.Type }}\n" +
			"Contact Value: {{ .Contact.Value }}"

		Convey("Test with nil data", func() {
			webhookPopulater := NewWebhookBodyPopulater(nil)

			actual, err := webhookPopulater.Populate(template)
			So(err, ShouldNotBeNil)
			So(actual, ShouldResemble, template)
		})

		Convey("Test with empty data", func() {
			webhookPopulater := NewWebhookBodyPopulater(&Contact{})
			expected := "" +
				"Contact Type: \n" +
				"Contact Value:"

			actual, err := webhookPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("Test with empty value", func() {
			webhookPopulater := NewWebhookBodyPopulater(&Contact{
				Type: "slack",
			})
			expected := "" +
				"Contact Type: slack\n" +
				"Contact Value:"

			actual, err := webhookPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("Test with empty type", func() {
			webhookPopulater := NewWebhookBodyPopulater(&Contact{
				Value: "#test_channel",
			})
			expected := "" +
				"Contact Type: \n" +
				"Contact Value: #test_channel"

			actual, err := webhookPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("Test with full template contact info", func() {
			webhookPopulater := NewWebhookBodyPopulater(&Contact{
				Type:  "slack",
				Value: "#test_channel",
			})
			expected := "" +
				"Contact Type: slack\n" +
				"Contact Value: #test_channel"

			actual, err := webhookPopulater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})
	})
}

func Test_TemplateWebhookDeliveryCheckURL(t *testing.T) {
	Convey("Test populating webhook delivery check url template", t, func() {
		template := "https://example.url/delivery/check/{{ .Contact.Type }}/{{ .SendAlertResponse.requestID }}/{{ .TriggerID }}"

		Convey("with nil data", func() {
			populater := NewWebhookDeliveryCheckURLPopulater(
				nil, nil, "")

			actual, err := populater.Populate(template)
			So(err, ShouldNotBeNil)
			So(actual, ShouldResemble, template)
		})

		Convey("with empty data", func() {
			populater := NewWebhookDeliveryCheckURLPopulater(
				&Contact{}, map[string]interface{}{}, "")

			expected := "https://example.url/delivery/check///"

			actual, err := populater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("with filled data and str requestID", func() {
			populater := NewWebhookDeliveryCheckURLPopulater(
				&Contact{
					Type:  "slack",
					Value: "some_value",
				},
				map[string]interface{}{
					"requestID": "test_id",
				},
				"")

			expected := "https://example.url/delivery/check/slack/test_id/"

			actual, err := populater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("with filled data and number in requestID", func() {
			populater := NewWebhookDeliveryCheckURLPopulater(
				&Contact{
					Type:  "slack",
					Value: "some_value",
				},
				map[string]interface{}{
					"requestID": 125,
				},
				"some_trigger_id")

			expected := "https://example.url/delivery/check/slack/125/some_trigger_id"

			actual, err := populater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})
	})
}

func Test_TemplateWebhookDeliveryCheckState(t *testing.T) {
	Convey("Test populating webhook delivery check template", t, func() {
		template := `{{ .Contact.Type }}/{{ .Contact.Value }}/{{ .DeliveryCheckResponse.some_field }}/{{ .TriggerID }}/{{ .StateConstants.DeliveryStateOK }}`

		Convey("with nil values", func() {
			populater := NewWebhookDeliveryCheckPopulater(nil, nil, "")

			actual, err := populater.Populate(template)
			So(err, ShouldNotBeNil)
			So(actual, ShouldResemble, template)
		})

		Convey("with empty values", func() {
			populater := NewWebhookDeliveryCheckPopulater(&Contact{}, map[string]interface{}{}, "")

			expected := "////OK"

			actual, err := populater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("with filled values", func() {
			populater := NewWebhookDeliveryCheckPopulater(
				&Contact{
					Type:  "slack",
					Value: "some_value",
				},
				map[string]interface{}{
					"some_field": 10,
				},
				"some_trigger_id")

			expected := "slack/some_value/10/some_trigger_id/OK"

			actual, err := populater.Populate(template)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})
	})
}
