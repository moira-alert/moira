package redis

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

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

	Convey("Trigger manipulation", t, func() {

		Convey("Test trigger has subscriptions with AnyTag is true", func() {
			trigger := &triggers[0]
			subscription := moira.SubscriptionData{
				ID:                "subscriptionID-00000000000001",
				Enabled:           true,
				Tags:              []string{tag1, tag2, tag3},
				Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
				ThrottlingEnabled: true,
				User:              user1,
			}
			subscription.AnyTags = true

			err := dataBase.SaveSubscription(&subscription)
			So(err, ShouldBeNil)

			hasSubscriptions, err := dataBase.triggerHasSubscriptions(trigger)
			So(err, ShouldBeNil)
			So(hasSubscriptions, ShouldBeTrue)

			err = dataBase.RemoveSubscription(subscription.ID)
			So(err, ShouldBeNil)

			hasSubscriptions, err = dataBase.triggerHasSubscriptions(trigger)
			So(err, ShouldBeNil)
			So(hasSubscriptions, ShouldBeFalse)
		})

		Convey("Test save-get-remove", func() {
			trigger := &triggers[0]

			//Check for not existing not written trigger
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

			ids, err := dataBase.GetLocalTriggerIDs()
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

			ids, err = dataBase.GetLocalTriggerIDs()
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
				Trigger:   trigger,
				LastCheck: moira.CheckData{},
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

		Convey("Save trigger with metrics and get metrics", func() {
			pattern1 := "my.test.*.metric*"
			metric1 := "my.test.super.metric1"

			pattern2 := "my.new.test.*.metric*"
			metric2 := "my.new.test.super.metric2"

			triggerVer1 := &moira.Trigger{
				ID:           "test-triggerID-id1",
				Name:         "test trigger 1 v1.0",
				Targets:      []string{pattern1},
				Tags:         []string{"test-tag-1"},
				Patterns:     []string{pattern1},
				TriggerType:  moira.RisingTrigger,
				AloneMetrics: map[string]bool{},
			}

			triggerVer2 := &moira.Trigger{
				ID:           "test-triggerID-id1",
				Name:         "test trigger 1 v2.0",
				Targets:      []string{pattern2},
				Tags:         []string{"test-tag-1"},
				Patterns:     []string{pattern2},
				TriggerType:  moira.RisingTrigger,
				AloneMetrics: map[string]bool{},
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
			So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err := dataBase.GetTrigger(triggerVer1.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *triggerVer1)

			ids, err := dataBase.GetLocalTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer1.ID})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer1.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer1.ID})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer1.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer1.ID})

			actualTriggers, err := dataBase.GetTriggers(ids)
			So(err, ShouldBeNil)
			So(actualTriggers, ShouldResemble, []*moira.Trigger{triggerVer1})

			//Save metrics
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val1})
			So(err, ShouldBeNil)

			//And check it
			actualValues, err := dataBase.GetMetricsValues([]string{metric1}, 0, 100)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {
				&moira.MetricValue{
					Timestamp:          val1.Timestamp,
					RetentionTimestamp: val1.RetentionTimestamp,
					Value:              val1.Value}}})

			actualPatternMetrics, err := dataBase.GetPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{metric1})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{})

			//Update trigger, change its pattern
			err = dataBase.SaveTrigger(triggerVer2.ID, triggerVer2)
			So(err, ShouldBeNil)

			//And check for existing by several pointers like id or tag
			actual, err = dataBase.GetTrigger(triggerVer2.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *triggerVer2)

			ids, err = dataBase.GetLocalTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer2.ID})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer2.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer2.ID})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer2.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{triggerVer2.ID})

			actualTriggers, err = dataBase.GetTriggers(ids)
			So(err, ShouldBeNil)
			So(actualTriggers, ShouldResemble, []*moira.Trigger{triggerVer2})

			//Save metrics for a new pattern metrics
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: val2})
			So(err, ShouldBeNil)

			//And check it
			actualValues, err = dataBase.GetMetricsValues([]string{metric2}, 0, 100)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric2: {
				&moira.MetricValue{
					Timestamp:          val2.Timestamp,
					RetentionTimestamp: val2.RetentionTimestamp,
					Value:              val2.Value}}})

			//And check old metrics, it must be empty
			actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 100)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{metric2})

			//It's time to remove trigger and check all data
			err = dataBase.RemoveTrigger(triggerVer2.ID)
			So(err, ShouldBeNil)

			actual, err = dataBase.GetTrigger(triggerVer2.ID)
			So(err, ShouldResemble, database.ErrNil)
			So(actual, ShouldResemble, moira.Trigger{})

			ids, err = dataBase.GetLocalTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})

			ids, err = dataBase.GetTagTriggerIDs(triggerVer2.Tags[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})

			ids, err = dataBase.GetPatternTriggerIDs(triggerVer2.Patterns[0])
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})

			actualTriggers, err = dataBase.GetTriggers(ids)
			So(err, ShouldBeNil)
			So(actualTriggers, ShouldResemble, []*moira.Trigger{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{})

			actualPatternMetrics, err = dataBase.GetPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			So(actualPatternMetrics, ShouldResemble, []string{})
		})

		Convey("Test trigger manipulations update 'triggers to reindex' list", func() {
			dataBase.flush()
			trigger := &triggers[0]

			err := dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)

			actualTrigger, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actualTrigger, ShouldResemble, *trigger)

			actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{trigger.ID})

			// Now update trigger
			trigger = &triggers[1]

			err = dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{trigger.ID})

			// Add new trigger
			trigger = &triggers[5]

			err = dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			So(err, ShouldBeNil)
			So(actual, ShouldHaveLength, 2)

			// Clean reindex list
			err = dataBase.RemoveTriggersToReindex(time.Now().Unix() + 1)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 10)
			So(err, ShouldBeNil)
			So(actual, ShouldBeEmpty)

			// Remove trigger
			err = dataBase.RemoveTrigger(trigger.ID)
			So(err, ShouldBeNil)

			actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix() - 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []string{trigger.ID})
		})
	})
}

func TestRemoteTrigger(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	pattern := "test.pattern.remote1"
	trigger := &moira.Trigger{
		ID:           "triggerID-0000000000010",
		Name:         "remote",
		Targets:      []string{"test.target.remote1"},
		Patterns:     []string{pattern},
		IsRemote:     true,
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
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
		Convey("Trigger should not be added to local triggers collection", func() {
			ids, err := dataBase.GetLocalTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger should be added to remote triggers collection", func() {
			ids, err := dataBase.GetRemoteTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should not be added to patterns collection", func() {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger pattern shouldn't be in patterns collection", func() {
			patterns, err := dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(patterns, ShouldResemble, []string{})
		})
	})

	Convey("Update remote trigger as local", t, func() {
		trigger.IsRemote = false
		trigger.Patterns = []string{pattern}
		Convey("Trigger should be saved correctly", func() {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be added to triggers collection", func() {
			ids, err := dataBase.GetLocalTriggerIDs()
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
		Convey("Trigger shouldn't be returned as local", func() {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to patterns collection", func() {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger pattern should be in patterns collection", func() {
			patterns, err := dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(patterns, ShouldResemble, trigger.Patterns)
		})

		trigger.IsRemote = true
		Convey("Update this trigger as remote", func() {
			err := dataBase.SaveTrigger(trigger.ID, trigger)
			So(err, ShouldBeNil)
			actual, err := dataBase.GetTrigger(trigger.ID)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, *trigger)
		})
		Convey("Trigger should be deleted from local triggers collection", func() {
			ids, err := dataBase.GetLocalTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger should still be in all triggers collection", func() {
			ids, err := dataBase.GetAllTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should be added to remote triggers collection", func() {
			ids, err := dataBase.GetRemoteTriggerIDs()
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{trigger.ID})
		})
		Convey("Trigger should deleted from patterns collection", func() {
			ids, err := dataBase.GetPatternTriggerIDs(pattern)
			So(err, ShouldBeNil)
			So(ids, ShouldResemble, []string{})
		})
		Convey("Trigger pattern should not be in patterns collection", func() {
			patterns, err := dataBase.GetPatterns()
			So(err, ShouldBeNil)
			So(patterns, ShouldResemble, []string{})
		})
	})
}

func TestTriggerErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual, err := dataBase.GetLocalTriggerIDs()
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
		ID:           "triggerID-0000000000001",
		Name:         "test trigger 1 v1.0",
		Targets:      []string{"test.target.1"},
		Tags:         []string{"test-tag-1"},
		Patterns:     []string{"test.pattern.1"},
		TriggerType:  moira.RisingTrigger,
		TTLState:     &moira.TTLStateNODATA,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000001",
		Name:         "test trigger 1 v2.0",
		Targets:      []string{"test.target.1", "test.target.2"},
		Tags:         []string{"test-tag-2", "test-tag-1"},
		Patterns:     []string{"test.pattern.2", "test.pattern.1"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{"t2": true},
	},
	{
		ID:           "triggerID-0000000000001",
		Name:         "test trigger 1 v3.0",
		Targets:      []string{"test.target.3"},
		Tags:         []string{"test-tag-2", "test-tag-3"},
		Patterns:     []string{"test.pattern.3", "test.pattern.2"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000004",
		Name:         "test trigger 4",
		Targets:      []string{"test.target.4"},
		Tags:         []string{"test-tag-4"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000005",
		Name:         "test trigger 5 (nobody is subscribed)",
		Targets:      []string{"test.target.5"},
		Tags:         []string{"test-tag-nosub"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000006",
		Name:         "test trigger 6 (throttling disabled)",
		Targets:      []string{"test.target.6"},
		Tags:         []string{"test-tag-throttling-disabled"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000007",
		Name:         "test trigger 7 (multiple subscribers)",
		Targets:      []string{"test.target.7"},
		Tags:         []string{"test-tag-multiple-subs"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000008",
		Name:         "test trigger 8 (duplicated contacts)",
		Targets:      []string{"test.target.8"},
		Tags:         []string{"test-tag-dup-contacts"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
	{
		ID:           "triggerID-0000000000009",
		Name:         "test trigger 9 (pseudo tag)",
		Targets:      []string{"test.target.9"},
		Tags:         []string{"test-degradation"},
		TriggerType:  moira.RisingTrigger,
		AloneMetrics: map[string]bool{},
	},
}
