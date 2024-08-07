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
