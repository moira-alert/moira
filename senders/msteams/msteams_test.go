package msteams

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/h2non/gock.v1"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Init tests", t, func() {
		sender := Sender{}
		senderSettings := map[string]string{
			"max_events": "-1",
		}
		Convey("Empty map should fail", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldNotResemble, nil)
		})
		Convey("Minimal settings", func() {
			err := sender.Init(senderSettings, logger, nil, "")
			So(err, ShouldResemble, nil)
			So(sender, ShouldNotResemble, Sender{})
			So(sender.maxEvents, ShouldResemble, -1)
		})
	})
}

func TestMSTeamsHttpResponse(t *testing.T) {
	sender := Sender{}
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	location, _ := time.LoadLocation("UTC")
	_ = sender.Init(map[string]string{
		"max_events": "-1",
	}, logger, location, "")
	event := moira.NotificationEvent{
		TriggerID: "TriggerID",
		Values:    map[string]float64{"t1": 123},
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

	Convey("When HTTP Response", t, func() {
		Convey("is 200 and body is '1' there should be no error", func() {
			defer gock.Off()
			gock.New("https://outlook.office.com/webhook/foo").
				Post("/").
				Reply(200).
				BodyString("1")
			contact := moira.ContactData{Value: "https://outlook.office.com/webhook/foo"}
			err := sender.SendEvents([]moira.NotificationEvent{event}, contact, trigger, make([][]byte, 0, 1), false)
			So(err, ShouldResemble, nil)
			So(gock.IsDone(), ShouldBeTrue)
		})
		Convey("is 200 and body is not '1', result should be an error", func() {
			defer gock.Off()
			gock.New("https://outlook.office.com/webhook/foo").
				Post("/").
				Reply(200).
				BodyString("Some error")
			contact := moira.ContactData{Value: "https://outlook.office.com/webhook/foo"}
			err := sender.SendEvents([]moira.NotificationEvent{event}, contact, trigger, make([][]byte, 0, 1), false)
			So(err.Error(), ShouldResemble, "teams endpoint responded with an error: Some error")
			So(gock.IsDone(), ShouldBeTrue)
		})
		Convey("is not any of HTTP success, result should be an error", func() {
			defer gock.Off()
			gock.New("https://outlook.office.com/webhook/foo").
				Post("/").
				Reply(500).
				BodyString("Some error")
			contact := moira.ContactData{Value: "https://outlook.office.com/webhook/foo"}
			err := sender.SendEvents([]moira.NotificationEvent{event}, contact, trigger, make([][]byte, 0, 1), false)
			So(err.Error(), ShouldResemble, "server responded with a non 2xx code: 500")
			So(gock.IsDone(), ShouldBeTrue)
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
	sender := Sender{location: location, maxEvents: -1, frontURI: "http://moira.url"}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
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

		Convey("Card uses the correct colour for subject state", func() {
			Convey("State is Red for Error", func() {
				actual := sender.buildMessage([]moira.NotificationEvent{{
					TriggerID: "TriggerID",
					Values:    map[string]float64{"t1": 123},
					Timestamp: 150000000,
					Metric:    "Metric",
					OldState:  moira.StateOK,
					State:     moira.StateERROR,
					Message:   nil,
				}}, moira.TriggerData{Name: "Name"}, false)
				expected := MessageCard{
					Context:     "http://schema.org/extensions",
					MessageType: "MessageCard",
					Summary:     "Moira Alert",
					ThemeColor:  Red,
					Title:       "ERROR Name",
					Sections: []Section{
						{
							ActivityTitle: "Description",
							ActivityText:  "",
							Facts: []Fact{
								{
									Name:  "02:40",
									Value: "```Metric = t1:123 (OK to ERROR)```",
								},
							},
						},
					},
				}
				So(actual, ShouldResemble, expected)
			})
			Convey("State is Orange for Warning", func() {
				actual := sender.buildMessage([]moira.NotificationEvent{{
					TriggerID: "TriggerID",
					Values:    map[string]float64{"t1": 123},
					Timestamp: 150000000,
					Metric:    "Metric",
					OldState:  moira.StateOK,
					State:     moira.StateWARN,
					Message:   nil,
				}}, moira.TriggerData{Name: "Name"}, false)
				expected := MessageCard{
					Context:     "http://schema.org/extensions",
					MessageType: "MessageCard",
					Summary:     "Moira Alert",
					ThemeColor:  Orange,
					Title:       "WARN Name",
					Sections: []Section{
						{
							ActivityTitle: "Description",
							ActivityText:  "",
							Facts: []Fact{
								{
									Name:  "02:40",
									Value: "```Metric = t1:123 (OK to WARN)```",
								},
							},
						},
					},
				}
				So(actual, ShouldResemble, expected)
			})
			Convey("State is Green for OK", func() {
				actual := sender.buildMessage([]moira.NotificationEvent{{
					TriggerID: "TriggerID",
					Values:    map[string]float64{"t1": 123},
					Timestamp: 150000000,
					Metric:    "Metric",
					OldState:  moira.StateWARN,
					State:     moira.StateOK,
					Message:   nil,
				}}, moira.TriggerData{Name: "Name"}, false)
				expected := MessageCard{
					Context:     "http://schema.org/extensions",
					MessageType: "MessageCard",
					Summary:     "Moira Alert",
					ThemeColor:  Green,
					Title:       "OK Name",
					Sections: []Section{
						{
							ActivityTitle: "Description",
							ActivityText:  "",
							Facts: []Fact{
								{
									Name:  "02:40",
									Value: "```Metric = t1:123 (WARN to OK)```",
								},
							},
						},
					},
				}
				So(actual, ShouldResemble, expected)
			})
			Convey("State is Black for NODATA", func() {
				actual := sender.buildMessage([]moira.NotificationEvent{{
					TriggerID: "TriggerID",
					Values:    map[string]float64{"t1": 123},
					Timestamp: 150000000,
					Metric:    "Metric",
					OldState:  moira.StateNODATA,
					State:     moira.StateNODATA,
					Message:   nil,
				}}, moira.TriggerData{Name: "Name"}, false)
				expected := MessageCard{
					Context:     "http://schema.org/extensions",
					MessageType: "MessageCard",
					Summary:     "Moira Alert",
					ThemeColor:  Black,
					Title:       "NODATA Name",
					Sections: []Section{
						{
							ActivityTitle: "Description",
							ActivityText:  "",
							Facts: []Fact{
								{
									Name:  "02:40",
									Value: "```Metric = t1:123 (NODATA to NODATA)```",
								},
							},
						},
					},
				}
				So(actual, ShouldResemble, expected)
			})

		})

		Convey("Create MessageCard with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := MessageCard{
				Context:     "http://schema.org/extensions",
				MessageType: "MessageCard",
				Summary:     "Moira Alert",
				ThemeColor:  Black,
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
						},
					},
				},
				PotentialAction: []Action{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenURITarget{
							{
								Os:  "default",
								URI: "http://moira.url/trigger/TriggerID",
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
				ThemeColor:  Black,
				Title:       "NODATA",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
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
				ThemeColor:  Black,
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "Warning",
								Value: "Please, *fix your system or tune this trigger* to generate less events.",
							},
						},
					},
				},
				PotentialAction: []Action{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenURITarget{
							{
								Os:  "default",
								URI: "http://moira.url/trigger/TriggerID",
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
				ThemeColor:  Black,
				Title:       "NODATA Name [tag1][tag2]",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "<h1>header1</h1>\n\n<p>some text <strong>bold text</strong></p>\n\n<h2>header 2</h2>\n\n<p>some other text <em>italic text</em></p>\n",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
							},
						},
					},
				},
				PotentialAction: []Action{
					{
						Type: "OpenUri",
						Name: "View in Moira",
						Targets: []OpenURITarget{
							{
								Os:  "default",
								URI: "http://moira.url/trigger/TriggerID",
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
				ThemeColor:  Black,
				Title:       "NODATA Name",
				Sections: []Section{
					{
						ActivityTitle: "Description",
						ActivityText:  "",
						Facts: []Fact{
							{
								Name:  "02:40",
								Value: "```Metric = t1:123 (OK to NODATA)```",
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
