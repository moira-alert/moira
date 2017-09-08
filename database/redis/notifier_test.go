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
*/
