package redis

import (
	"testing"
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func TestTriggerStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Trigger manipulation", t, func() {
		Convey("Test save-get-remove", func() {
			trigger := &triggers[0]

			//Check for not existing not writen trigger
			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.Trigger{})

			err = dataBase.RemoveTrigger(trigger.ID)
			So(err, ShouldBeNil)

			//Now write it
			err = dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *trigger)

			ids, err := dataBase.GetTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(trigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})

			actualTriggers, err := dataBase.GetTriggers(ids)
			So(err, ShouldBeNil)
			So(actualTriggers, ShouldResemble, []*moira.Trigger{trigger})

			//Also we write new patterns
			actualPatterns, err := dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(actualPatterns, ShouldResemble, trigger.Patterns)

			//And tags
			actualTags, err := dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(actualTags, ShouldResemble, trigger.Tags)

			//Now just add tag and pattern in trigger and save it
			trigger = nil
			changedTrigger := &triggers[1]
			err = dataBase.SaveTrigger(changedTrigger.ID, changedTrigger)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(changedTrigger.ID)
			So(err, ShouldBeNil)
			So(actual.Name, ShouldResemble, changedTrigger.Name)

			//Now we can get this trigger by two tags
			ids, err = dataBase.GetTagTriggerIDs(changedTrigger.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedTrigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(changedTrigger.Tags[1])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedTrigger.ID})

			//And we have new tag in tags list
			actualTags, err = dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(actualTags, ShouldHaveLength, 2)

			//Also we can get this trigger by new pattern
			ids, err = dataBase.GetPatternTriggerIDs(changedTrigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedTrigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(changedTrigger.Patterns[1])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedTrigger.ID})

			//And we have new pattern in patterns list
			actualPatterns, err = dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(actualPatterns, ShouldHaveLength, 2)

			//Now remove old tag and pattern in trigger and save it
			oldTag := changedTrigger.Tags[1]
			oldPattern := changedTrigger.Patterns[1]
			changedTrigger = nil
			changedAgainTrigger := &triggers[2]
			err = dataBase.SaveTrigger(changedAgainTrigger.ID, changedAgainTrigger)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(changedAgainTrigger.ID)
			So(err, ShouldBeNil)
			So(actual.Name, ShouldResemble, changedAgainTrigger.Name)

			//Now we can't find trigger by old tag but can get it by new one tag
			ids, err = dataBase.GetTagTriggerIDs(oldTag)
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[1])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			//But we still has this tag in tags list with new one
			actualTags, err = dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(actualTags, ShouldHaveLength, 3)

			//Same story like tags and trigger with pattern and trigger
			ids, err = dataBase.GetPatternTriggerIDs(oldPattern)
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[1])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			//But this pattern no more in pattern list, it is not needed
			actualTags, err = dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(actualTags, ShouldHaveLength, 2)

			//Stop it!! Remove trigger and check for no existing it by pointers
			err = dataBase.RemoveTrigger(changedAgainTrigger.ID)
			So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(changedAgainTrigger.ID)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.Trigger{})

			ids, err = dataBase.GetTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[1])
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[1])
			So(err, ShouldBeNil)
			So(ids, ShouldBeEmpty)

			actualTriggers, err = dataBase.GetTriggers([]string{changedAgainTrigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggers, ShouldResemble, []*moira.Trigger{nil})

			//Also we delete all patterns
			actualPatterns, err = dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(actualPatterns, ShouldBeEmpty)

			//But has all tags
			actualTags, err = dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(actualTags, ShouldHaveLength, 3)
		})

		Convey("Save trigger with lastCheck and throttling and GetTriggerChecks", func() {
			trigger := triggers[5]
			triggerCheck := &moira.TriggerCheck{
				Trigger: trigger,
			}

			err := dataBase.SaveTrigger(trigger.ID, &trigger)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, trigger)

			actualTriggerChecks, err := dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Add check data
			err = dataBase.SetTriggerLastCheck(trigger.ID, &lastCheckTest, false)
			So(err, ShouldBeNil)

			triggerCheck.LastCheck = lastCheckTest
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//And throttling
			err = dataBase.SetTriggerThrottling(trigger.ID, time.Now().Add(-time.Minute))
			So(err, ShouldBeNil)

			//But it is foul
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Now good throttling
			th := time.Now().Add(time.Minute)
			err = dataBase.SetTriggerThrottling(trigger.ID, th)
			So(err, ShouldBeNil)

			triggerCheck.Throttling = th.Unix()
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Remove throttling
			err = dataBase.DeleteTriggerThrottling(trigger.ID)
			So(err, ShouldBeNil)

			triggerCheck.Throttling = 0
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Can not remove check data, but can remove trigger!
			err = dataBase.RemoveTrigger(trigger.ID)
			So(err, ShouldBeNil)

			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			So(err, ShouldBeNil)
			So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{nil})
		})

	})
}

func TestRemoteTrigger(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	trigger := &moira.Trigger{
		ID:       "triggerID-0000000000010",
		Name:     "remote",
		Targets:  []string{"test.target.remote1"},
		Patterns: []string{"test.pattern.remote1"},
		IsRemote: true,
	}
	dataBase.flush()
	defer dataBase.flush()

	Convey("Saving remote trigger", t, func() {
		Convey("Trigger should be saved correctly", func() {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be added to triggers collection", func() {
			ids, err := dataBase.GetAllTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should not be returned as non-remote trigger", func() { //TODO rewrite msg
			ids, err := dataBase.GetTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger should be added to remote triggers collection", func() {
			ids, err := dataBase.GetRemoteTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should not be added to patterns collection", func() {
			ids, err := dataBase.GetPatternTriggerIDs(trigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
	})

	Convey("Resaving remote trigger as non-remote", t, func() {
		trigger.IsRemote = false
		Convey("Trigger should be saved correctly", func() {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be added to triggers collection", func() {
			ids, err := dataBase.GetTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to all triggers collection", func() {
			ids, err := dataBase.GetAllTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger shouldn't be added to remote triggers collection", func() {
			ids, err := dataBase.GetRemoteTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger shouldn't be returned as non-remote", func() {
			ids, err := dataBase.GetPatternTriggerIDs(trigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to patterns collection", func() {
			ids, err := dataBase.GetPatternTriggerIDs(trigger.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
	})
}

func TestTriggerErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual, err := dataBase.GetTriggerIDs()
		So(err, ShouldNotBeNil)
		So(actual, ShouldBeNil)

		actual1, err := dataBase.GetTrigger("")
		So(err, ShouldNotBeNil)
		So(actual1, ShouldResemble, moira.Trigger{})

		actual2, err := dataBase.GetTriggers([]string{})
		So(err, ShouldNotBeNil)
		So(actual2, ShouldBeNil)

		actual3, err := dataBase.GetTriggerChecks([]string{})
		So(err, ShouldNotBeNil)
		So(actual3, ShouldBeNil)

		err = dataBase.SaveTrigger("", &triggers[0])
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveTrigger("")
		So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetPatternTriggerIDs("")
		So(err, ShouldNotBeNil)
		So(actual4, ShouldBeNil)

		err = dataBase.RemovePatternTriggerIDs("")
		So(err, ShouldNotBeNil)
	})
}

var triggers = []moira.Trigger{
	{
		ID:       "triggerID-0000000000001",
		Name:     "test trigger 1 v1.0",
		Targets:  []string{"test.target.1"},
		Tags:     []string{"test-tag-1"},
		Patterns: []string{"test.pattern.1"},
	},
	{
		ID:       "triggerID-0000000000001",
		Name:     "test trigger 1 v2.0",
		Targets:  []string{"test.target.1", "test.target.2"},
		Tags:     []string{"test-tag-2", "test-tag-1"},
		Patterns: []string{"test.pattern.2", "test.pattern.1"},
	},
	{
		ID:       "triggerID-0000000000001",
		Name:     "test trigger 1 v3.0",
		Targets:  []string{"test.target.3"},
		Tags:     []string{"test-tag-2", "test-tag-3"},
		Patterns: []string{"test.pattern.3", "test.pattern.2"},
	},
	{
		ID:      "triggerID-0000000000004",
		Name:    "test trigger 4",
		Targets: []string{"test.target.4"},
		Tags:    []string{"test-tag-4"},
	},
	{
		ID:      "triggerID-0000000000005",
		Name:    "test trigger 5 (nobody is subscribed)",
		Targets: []string{"test.target.5"},
		Tags:    []string{"test-tag-nosub"},
	},
	{
		ID:      "triggerID-0000000000006",
		Name:    "test trigger 6 (throttling disabled)",
		Targets: []string{"test.target.6"},
		Tags:    []string{"test-tag-throttling-disabled"},
	},
	{
		ID:      "triggerID-0000000000007",
		Name:    "test trigger 7 (multiple subscribers)",
		Targets: []string{"test.target.7"},
		Tags:    []string{"test-tag-multiple-subs"},
	},
	{
		ID:      "triggerID-0000000000008",
		Name:    "test trigger 8 (duplicated contacts)",
		Targets: []string{"test.target.8"},
		Tags:    []string{"test-tag-dup-contacts"},
	},
	{
		ID:      "triggerID-0000000000009",
		Name:    "test trigger 9 (pseudo tag)",
		Targets: []string{"test.target.9"},
		Tags:    []string{"test-degradation"},
	},
}
