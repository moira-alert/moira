// nolint
package redis

import (
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNotifierDataBase(t *testing.T) {
	config := Config{}
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, Config{Port: "6379", Host: "localhost"})

	Convey("Try get trigger by empty id, should be error", t, func() {
		db := NewDatabase(logger, config)
		db.pool = dataBase.pool
		_, err := db.GetTrigger("")
		So(err, ShouldNotBeEmpty)
	})
}

/*
var triggers = []moira.TriggerData{
	{
		ID:         "triggerID-0000000000001",
		Name:       "test trigger 1",
		Targets:    []string{"test.target.1"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-1"},
	},
	{
		ID:         "triggerID-0000000000002",
		Name:       "test trigger 2",
		Targets:    []string{"test.target.2"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-2"},
	},
	{
		ID:         "triggerID-0000000000003",
		Name:       "test trigger 3",
		Targets:    []string{"test.target.3"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-3"},
	},
	{
		ID:         "triggerID-0000000000004",
		Name:       "test trigger 4",
		Targets:    []string{"test.target.4"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-4"},
	},
	{
		ID:         "triggerID-0000000000005",
		Name:       "test trigger 5 (nobody is subscribed)",
		Targets:    []string{"test.target.5"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-nosub"},
	},
	{
		ID:         "triggerID-0000000000006",
		Name:       "test trigger 6 (throttling disabled)",
		Targets:    []string{"test.target.6"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-throttling-disabled"},
	},
	{
		ID:         "triggerID-0000000000007",
		Name:       "test trigger 7 (multiple subscribers)",
		Targets:    []string{"test.target.7"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-multiple-subs"},
	},
	{
		ID:         "triggerID-0000000000008",
		Name:       "test trigger 8 (duplicated contacts)",
		Targets:    []string{"test.target.8"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-dup-contacts"},
	},
	{
		ID:         "triggerID-0000000000009",
		Name:       "test trigger 9 (pseudo tag)",
		Targets:    []string{"test.target.9"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-degradation"},
	},
}

var subscriptions = []moira.SubscriptionData{
	{
		ID:                "subscriptionID-00000000000001",
		Enabled:           true,
		Tags:              []string{"test-tag-1"},
		Contacts:          []string{contacts[0].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:       "subscriptionID-00000000000002",
		Enabled:  true,
		Tags:     []string{"test-tag-2"},
		Contacts: []string{contacts[1].ID},
		Schedule: moira.ScheduleData{
			StartOffset:    10,
			EndOffset:      20,
			TimezoneOffset: 0,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: true}, // Tuesday 00:10 - 00:20
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:       "subscriptionID-00000000000003",
		Enabled:  true,
		Tags:     []string{"test-tag-3"},
		Contacts: []string{contacts[2].ID},
		Schedule: moira.ScheduleData{
			StartOffset:    0,   // 0:00 (GMT +5) after
			EndOffset:      900, // 15:00 (GMT +5)
			TimezoneOffset: -300,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: false},
				{Enabled: true},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:       "subscriptionID-00000000000004",
		Enabled:  true,
		Tags:     []string{"test-tag-4"},
		Contacts: []string{contacts[3].ID},
		Schedule: moira.ScheduleData{
			StartOffset:    660, // 16:00 (GMT +5) before
			EndOffset:      900, // 20:00 (GMT +5)
			TimezoneOffset: 0,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: false},
				{Enabled: true},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000005",
		Enabled:           false,
		Tags:              []string{"test-tag-1"},
		Contacts:          []string{contacts[0].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000006",
		Enabled:           false,
		Tags:              []string{"test-tag-slack"},
		Contacts:          []string{contacts[4].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000007",
		Enabled:           false,
		Tags:              []string{"unknown-contact-type"},
		Contacts:          []string{contacts[5].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000008",
		Enabled:           true,
		Tags:              []string{"test-tag-throttling-disabled"},
		Contacts:          []string{contacts[0].ID},
		ThrottlingEnabled: false,
	},
	{
		ID:       "subscriptionID-00000000000009",
		Enabled:  true,
		Tags:     []string{"test-tag-multiple-subs"},
		Contacts: []string{contacts[2].ID},
		Schedule: moira.ScheduleData{
			StartOffset:    0,   // 0:00 (GMT +5) after
			EndOffset:      900, // 15:00 (GMT +5)
			TimezoneOffset: -300,
			Days: []moira.ScheduleDataDay{
				{Enabled: false},
				{Enabled: false},
				{Enabled: true},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
				{Enabled: false},
			},
		},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000010",
		Enabled:           true,
		Tags:              []string{"test-tag-multiple-subs"},
		Contacts:          []string{contacts[0].ID},
		ThrottlingEnabled: false,
	},
	{
		ID:                "subscriptionID-00000000000011",
		Enabled:           true,
		Tags:              []string{"test-tag-dup-contacts"},
		Contacts:          []string{contacts[6].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000012",
		Enabled:           true,
		Tags:              []string{"test-tag-dup-contacts"},
		Contacts:          []string{contacts[7].ID},
		ThrottlingEnabled: true,
	},
	{
		ID:                "subscriptionID-00000000000013",
		Enabled:           true,
		Tags:              []string{"degradation"},
		Contacts:          []string{contacts[0].ID},
		ThrottlingEnabled: false,
	},
}*/
