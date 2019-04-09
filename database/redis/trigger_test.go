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
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Trigger manipulation", t, func(c C) {
		Convey("Test save-get-remove", t, func(c C) {
			trigger := &triggers[0]

			//Check for not existing not writen trigger
			actual, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.Trigger{})

			err = dataBase.RemoveTrigger(trigger.ID)
			c.So(err, ShouldBeNil)

			//Now write it
			err = dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *trigger)

			ids, err := dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(trigger.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})

			actualTriggers, err := dataBase.GetTriggers(ids)
			c.So(err, ShouldBeNil)
			c.So(actualTriggers, ShouldResemble, []*moira.Trigger{trigger})

			//Also we write new patterns
			actualPatterns, err := dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(actualPatterns, ShouldResemble, trigger.Patterns)

			//And tags
			actualTags, err := dataBase.GetTagNames()
			c.So(err, ShouldBeNil)
			c.So(actualTags, ShouldResemble, trigger.Tags)

			//Now just add tag and pattern in trigger and save it
			trigger = nil
			changedTrigger := &triggers[1]
			err = dataBase.SaveTrigger(changedTrigger.ID, changedTrigger)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(changedTrigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual.Name, ShouldResemble, changedTrigger.Name)

			//Now we can get this trigger by two tags
			ids, err = dataBase.GetTagTriggerIDs(changedTrigger.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedTrigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(changedTrigger.Tags[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedTrigger.ID})

			//And we have new tag in tags list
			actualTags, err = dataBase.GetTagNames()
			c.So(err, ShouldBeNil)
			c.So(actualTags, ShouldHaveLength, 2)

			//Also we can get this trigger by new pattern
			ids, err = dataBase.GetPatternTriggerIDs(changedTrigger.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedTrigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(changedTrigger.Patterns[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedTrigger.ID})

			//And we have new pattern in patterns list
			actualPatterns, err = dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(actualPatterns, ShouldHaveLength, 2)

			//Now remove old tag and pattern in trigger and save it
			oldTag := changedTrigger.Tags[1]
			oldPattern := changedTrigger.Patterns[1]
			changedTrigger = nil
			changedAgainTrigger := &triggers[2]
			err = dataBase.SaveTrigger(changedAgainTrigger.ID, changedAgainTrigger)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(changedAgainTrigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual.Name, ShouldResemble, changedAgainTrigger.Name)

			//Now we can't find trigger by old tag but can get it by new one tag
			ids, err = dataBase.GetTagTriggerIDs(oldTag)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			//But we still has this tag in tags list with new one
			actualTags, err = dataBase.GetTagNames()
			c.So(err, ShouldBeNil)
			c.So(actualTags, ShouldHaveLength, 3)

			//Same story like tags and trigger with pattern and trigger
			ids, err = dataBase.GetPatternTriggerIDs(oldPattern)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{changedAgainTrigger.ID})

			//But this pattern no more in pattern list, it is not needed
			actualTags, err = dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(actualTags, ShouldHaveLength, 2)

			//Stop it!! Remove trigger and check for no existing it by pointers
			err = dataBase.RemoveTrigger(changedAgainTrigger.ID)
			c.So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(changedAgainTrigger.ID)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.Trigger{})

			ids, err = dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetTagTriggerIDs(changedAgainTrigger.Tags[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			ids, err = dataBase.GetPatternTriggerIDs(changedAgainTrigger.Patterns[1])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldBeEmpty)

			actualTriggers, err = dataBase.GetTriggers([]string{changedAgainTrigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggers, ShouldResemble, []*moira.Trigger{nil})

			//Also we delete all patterns
			actualPatterns, err = dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(actualPatterns, ShouldBeEmpty)

			//But has all tags
			actualTags, err = dataBase.GetTagNames()
			c.So(err, ShouldBeNil)
			c.So(actualTags, ShouldHaveLength, 3)
		})

		Convey("Save trigger with lastCheck and throttling and GetTriggerChecks", t, func(c C) {
			trigger := triggers[5]
			triggerCheck := &moira.TriggerCheck{
				Trigger: trigger,
			}

			err := dataBase.SaveTrigger(trigger.ID, &trigger)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, trigger)

			actualTriggerChecks, err := dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Add check data
			err = dataBase.SetTriggerLastCheck(trigger.ID, &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			triggerCheck.LastCheck = lastCheckTest
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//And throttling
			err = dataBase.SetTriggerThrottling(trigger.ID, time.Now().Add(-time.Minute))
			c.So(err, ShouldBeNil)

			//But it is foul
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Now good throttling
			th := time.Now().Add(time.Minute)
			err = dataBase.SetTriggerThrottling(trigger.ID, th)
			c.So(err, ShouldBeNil)

			triggerCheck.Throttling = th.Unix()
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Remove throttling
			err = dataBase.DeleteTriggerThrottling(trigger.ID)
			c.So(err, ShouldBeNil)

			triggerCheck.Throttling = 0
			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{triggerCheck})

			//Can not remove check data, but can remove trigger!
			err = dataBase.RemoveTrigger(trigger.ID)
			c.So(err, ShouldBeNil)

			actualTriggerChecks, err = dataBase.GetTriggerChecks([]string{trigger.ID})
			c.So(err, ShouldBeNil)
			c.So(actualTriggerChecks, ShouldResemble, []*moira.TriggerCheck{nil})
		})

		Convey("Save trigger with metrics and get metrics", t, func(c C) {
			pattern1 := "my.test.*.metric*"
			metric1 := "my.test.super.metric1"

			pattern2 := "my.new.test.*.metric*"
			metric2 := "my.new.test.super.metric2"

			triggerVer1 := &moira.Trigger{
				ID:          "test-triggerID-id1",
				Name:        "test trigger 1 v1.0",
				Targets:     []string{pattern1},
				Tags:        []string{"test-tag-1"},
				Patterns:    []string{pattern1},
				TriggerType: moira.RisingTrigger,
			}

			triggerVer2 := &moira.Trigger{
				ID:          "test-triggerID-id1",
				Name:        "test trigger 1 v2.0",
				Targets:     []string{pattern2},
				Tags:        []string{"test-tag-1"},
				Patterns:    []string{pattern2},
				TriggerType: moira.RisingTrigger,
			}

			val1 := &moira.MatchedMetric{
				Patterns:           []string{pattern1},
				Metric:             metric1,
				Retention:          10,
				RetentionTimestamp: 10,
				Timestamp:          15,
				Value:              1,
			}
			val2 := &moira.MatchedMetric{
				Patterns:           []string{pattern2},
				Metric:             metric2,
				Retention:          10,
				RetentionTimestamp: 20,
				Timestamp:          22,
				Value:              2,
			}

			//Add trigger
			err := dataBase.SaveTrigger(triggerVer1.ID, triggerVer1)
			c.So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err := dataBase.GetTrigger(triggerVer1.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *triggerVer1)

			ids, err := dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer1.ID})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer1.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer1.ID})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer1.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer1.ID})

			actualTriggers, err := dataBase.GetTriggers(ids)
			c.So(err, ShouldBeNil)
			c.So(actualTriggers, ShouldResemble, []*moira.Trigger{triggerVer1})

			//Save metrics
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val1})
			c.So(err, ShouldBeNil)

			//And check it
			actualValues, err := dataBase.GetMetricsValues([]string{metric1}, 0, 100)
			c.So(err, ShouldBeNil)
			c.So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {
				&moira.MetricValue{
					Timestamp:          val1.Timestamp,
					RetentionTimestamp: val1.RetentionTimestamp,
					Value:              val1.Value}}})

			actualPatternMetrics, err := dataBase.GetPatternMetrics(pattern1)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{metric1})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{})

			//Update trigger, change its pattern
			err = dataBase.SaveTrigger(triggerVer2.ID, triggerVer2)
			c.So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(triggerVer2.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *triggerVer2)

			ids, err = dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer2.ID})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer2.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer2.ID})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer2.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{triggerVer2.ID})

			actualTriggers, err = dataBase.GetTriggers(ids)
			c.So(err, ShouldBeNil)
			c.So(actualTriggers, ShouldResemble, []*moira.Trigger{triggerVer2})

			//Save metrics for a new pattern metrics
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: val2})
			c.So(err, ShouldBeNil)

			//And check it
			actualValues, err = dataBase.GetMetricsValues([]string{metric2}, 0, 100)
			c.So(err, ShouldBeNil)
			c.So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric2: {
				&moira.MetricValue{
					Timestamp:          val2.Timestamp,
					RetentionTimestamp: val2.RetentionTimestamp,
					Value:              val2.Value}}})

			//And check old metrics, it must be empty
			actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 100)
			c.So(err, ShouldBeNil)
			c.So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern1)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{metric2})

			//It's time to remove trigger and check all data
			err = dataBase.RemoveTrigger(triggerVer2.ID)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(triggerVer2.ID)
			c.So(err, ShouldResemble, database.ErrNil)
			c.So(actual, ShouldResemble, moira.Trigger{})

			ids, err = dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer2.Tags[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer2.Patterns[0])
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})

			actualTriggers, err = dataBase.GetTriggers(ids)
			c.So(err, ShouldBeNil)
			c.So(actualTriggers, ShouldResemble, []*moira.Trigger{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern1)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			c.So(err, ShouldBeNil)
			c.So(actualPatternMetrics, ShouldResemble, []string{})
		})

		Convey("Test trigger manipulations update 'triggers to reindex' list", t, func(c C) {
			dataBase.flush()
			trigger := &triggers[0]

			err := dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)

			actualTrigger, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actualTrigger, ShouldResemble, *trigger)

			actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{trigger.ID})

			// Now update trigger
			trigger = &triggers[1]

			err = dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{trigger.ID})

			// Add new trigger
			trigger = &triggers[5]

			err = dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldHaveLength, 2)

			// Clean reindex list
			err = dataBase.RemoveTriggersToReindex(time.Now().Unix() + 1)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldBeEmpty)

			// Remove trigger
			err = dataBase.RemoveTrigger(trigger.ID)
			c.So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []string{trigger.ID})
		})
	})
}

func TestRemoteTrigger(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	pattern := "test.pattern.remote1"
	trigger := &moira.Trigger{
		ID:          "triggerID-0000000000010",
		Name:        "remote",
		Targets:     []string{"test.target.remote1"},
		Patterns:    []string{pattern},
		IsRemote:    true,
		TriggerType: moira.RisingTrigger,
	}
	dataBase.flush()
	defer dataBase.flush()

	Convey("Saving remote trigger", t, func(c C) {
		Convey("Trigger should be saved correctly", t, func(c C) {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be added to triggers collection", t, func(c C) {
			ids, err := dataBase.GetAllTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should not be added to local triggers collection", t, func(c C) {
			ids, err := dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger should be added to remote triggers collection", t, func(c C) {
			ids, err := dataBase.GetRemoteTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should not be added to patterns collection", t, func(c C) {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger pattern shouldn't be in patterns collection", t, func(c C) {
			patterns, err := dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(patterns, ShouldResemble, []string{})
		})
	})

	Convey("Update remote trigger as local", t, func(c C) {
		trigger.IsRemote = false
		trigger.Patterns = []string{pattern}
		Convey("Trigger should be saved correctly", t, func(c C) {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be added to triggers collection", t, func(c C) {
			ids, err := dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to all triggers collection", t, func(c C) {
			ids, err := dataBase.GetAllTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger shouldn't be added to remote triggers collection", t, func(c C) {
			ids, err := dataBase.GetRemoteTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger shouldn't be returned as local", t, func(c C) {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to patterns collection", t, func(c C) {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger pattern should be in patterns collection", t, func(c C) {
			patterns, err := dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(patterns, ShouldResemble, trigger.Patterns)
		})

		trigger.IsRemote = true
		Convey("Update this trigger as remote", t, func(c C) {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			c.So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be deleted from local triggers collection", t, func(c C) {
			ids, err := dataBase.GetLocalTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger should still be in all triggers collection", t, func(c C) {
			ids, err := dataBase.GetAllTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to remote triggers collection", t, func(c C) {
			ids, err := dataBase.GetRemoteTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should deleted from patterns collection", t, func(c C) {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			c.So(err, ShouldBeNil)
			c.So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger pattern should not be in patterns collection", t, func(c C) {
			patterns, err := dataBase.GetPatterns()
			c.So(err, ShouldBeNil)
			c.So(patterns, ShouldResemble, []string{})
		})
	})
}

func TestTriggerErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		actual, err := dataBase.GetLocalTriggerIDs()
		c.So(err, ShouldNotBeNil)
		c.So(actual, ShouldBeNil)

		actual1, err := dataBase.GetTrigger("")
		c.So(err, ShouldNotBeNil)
		c.So(actual1, ShouldResemble, moira.Trigger{})

		actual2, err := dataBase.GetTriggers([]string{})
		c.So(err, ShouldNotBeNil)
		c.So(actual2, ShouldBeNil)

		actual3, err := dataBase.GetTriggerChecks([]string{})
		c.So(err, ShouldNotBeNil)
		c.So(actual3, ShouldBeNil)

		err = dataBase.SaveTrigger("", &triggers[0])
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveTrigger("")
		c.So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetPatternTriggerIDs("")
		c.So(err, ShouldNotBeNil)
		c.So(actual4, ShouldBeNil)

		err = dataBase.RemovePatternTriggerIDs("")
		c.So(err, ShouldNotBeNil)
	})
}

var triggers = []moira.Trigger{
	{
		ID:          "triggerID-0000000000001",
		Name:        "test trigger 1 v1.0",
		Targets:     []string{"test.target.1"},
		Tags:        []string{"test-tag-1"},
		Patterns:    []string{"test.pattern.1"},
		TriggerType: moira.RisingTrigger,
		TTLState:    &moira.TTLStateNODATA,
	},
	{
		ID:          "triggerID-0000000000001",
		Name:        "test trigger 1 v2.0",
		Targets:     []string{"test.target.1", "test.target.2"},
		Tags:        []string{"test-tag-2", "test-tag-1"},
		Patterns:    []string{"test.pattern.2", "test.pattern.1"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000001",
		Name:        "test trigger 1 v3.0",
		Targets:     []string{"test.target.3"},
		Tags:        []string{"test-tag-2", "test-tag-3"},
		Patterns:    []string{"test.pattern.3", "test.pattern.2"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000004",
		Name:        "test trigger 4",
		Targets:     []string{"test.target.4"},
		Tags:        []string{"test-tag-4"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000005",
		Name:        "test trigger 5 (nobody is subscribed)",
		Targets:     []string{"test.target.5"},
		Tags:        []string{"test-tag-nosub"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000006",
		Name:        "test trigger 6 (throttling disabled)",
		Targets:     []string{"test.target.6"},
		Tags:        []string{"test-tag-throttling-disabled"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000007",
		Name:        "test trigger 7 (multiple subscribers)",
		Targets:     []string{"test.target.7"},
		Tags:        []string{"test-tag-multiple-subs"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000008",
		Name:        "test trigger 8 (duplicated contacts)",
		Targets:     []string{"test.target.8"},
		Tags:        []string{"test-tag-dup-contacts"},
		TriggerType: moira.RisingTrigger,
	},
	{
		ID:          "triggerID-0000000000009",
		Name:        "test trigger 9 (pseudo tag)",
		Targets:     []string{"test.target.9"},
		Tags:        []string{"test-degradation"},
		TriggerType: moira.RisingTrigger,
	},
}
