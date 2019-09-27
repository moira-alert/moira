package msteams

import (
	"github.com/moira-alert/moira/logging/go-logging"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Init tests", t, func() {
		sender := Sender{}
		senderSettings := map[string]string{}
		Convey("Empty map", func() {
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldResemble, nil)
			So(sender, ShouldNotResemble, Sender{})
		})
	})
}

func TestValidWebhook(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	Convey("MS Teams Webhook validity", t, func() {
		Convey("https://outlook.office.com/webhook/foo is valid", func() {
			err := sender.isValidWebhookURL("https://outlook.office.com/webhook/foo")
			So(err, ShouldResemble, nil)
		})
		Convey("https://moira.url is invalid", func() {
			err := sender.isValidWebhookURL("https://moira.url")
			So(err, ShouldNotResemble, nil)
		})
	})
}

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(123)

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
			Message:   nil,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: `# header1
some text **bold text**
## header 2
some other text _italic text_`,
		}

		Convey("Create MessageCard with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  "000000",
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
						},
					},
				},
				PotentialAction: []Actions{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenUriTarget{
							{
								Os:  "default",
								Uri: "http://moira.url/trigger/TriggerID",
							},
						},
					},
				},
			}
			So(actual, ShouldResemble, expected)
		})

		Convey("Create MessageCard with empty trigger", func() {
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  "000000",
				Title:       "NODATA",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
						},
					},
				},
			}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			So(actual, ShouldResemble, expected)
		})

		Convey("Create MessageCard with one event and a throttle warning", func() {
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  "000000",
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "Warning",
								Value: "Please, *fix your system or tune this trigger* to generate less events.",
							},
						},
					},
				},
				PotentialAction: []Actions{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenUriTarget{
							{
								Os:  "default",
								Uri: "http://moira.url/trigger/TriggerID",
							},
						},
					},
				},
			}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			So(actual, ShouldResemble, expected)
		})

		Convey("Create MessageCard with 6 events", func() {
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  "000000",
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
						},
					},
				},
				PotentialAction: []Actions{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenUriTarget{
							{
								Os:  "default",
								Uri: "http://moira.url/trigger/TriggerID",
							},
						},
					},
				},
			}
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			So(actual, ShouldResemble, expected)
		})

		Convey("Create MessageCard without Trigger action, but with trigger name", func() {
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  "000000",
				Title:       "NODATA Name",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = 123 (OK to NODATA)```",
							},
						},
					},
				},
			}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			So(actual, ShouldResemble, expected)
		})
	})
}
