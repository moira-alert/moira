package redis

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnusedTriggers(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Check marking unused", t, func(c C) {
		// Check it before any trigger is marked unused
		triggerIDs, err := dataBase.GetUnusedTriggerIDs()
		c.So(err, ShouldBeNil)
		c.So(triggerIDs, ShouldBeEmpty)

		// Mark trigger 123 unused
		err = dataBase.MarkTriggersAsUnused("123")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"123"})
		c.So(err, ShouldBeNil)

		// Repeat procedure till success
		err = dataBase.MarkTriggersAsUnused("123")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"123"})
		c.So(err, ShouldBeNil)

		// Trying to unmark it
		err = dataBase.MarkTriggersAsUsed("123")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldBeEmpty)
		c.So(err, ShouldBeNil)

		// Ok, let's raise the rates
		err = dataBase.MarkTriggersAsUnused("123", "234", "345")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"123", "234", "345"})
		c.So(err, ShouldBeNil)

		// But, maybe we want to see the world burn?
		err = dataBase.MarkTriggersAsUnused("123", "234", "345")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"123", "234", "345"})
		c.So(err, ShouldBeNil)

		err = dataBase.MarkTriggersAsUsed("123", "234")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"345"})
		c.So(err, ShouldBeNil)

		// Okey, I want to unmark non-existing triggers
		err = dataBase.MarkTriggersAsUsed("alalala", "babababa")
		c.So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldResemble, []string{"345"})
		c.So(err, ShouldBeNil)

		// AAAAND magic
		err = dataBase.MarkTriggersAsUsed()
		c.So(err, ShouldBeNil)

		err = dataBase.MarkTriggersAsUnused()
		c.So(err, ShouldBeNil)
	})

	Convey("Check triggers are marked used and unused properly", t, func(c C) {
		trigger1Ver1 := &moira.Trigger{
			ID:          "triggerID-0000000000001",
			Name:        "test trigger 1 v1.0",
			Targets:     []string{"test.target.1"},
			Tags:        []string{"test-tag-1"},
			Patterns:    []string{"test.pattern.1"},
			TriggerType: moira.RisingTrigger,
		}

		trigger1Ver2 := &moira.Trigger{
			ID:          "triggerID-0000000000001",
			Name:        "test trigger 1 v2.0",
			Targets:     []string{"test.target.1"},
			Tags:        []string{"test-tag-2", "test-tag-1"},
			Patterns:    []string{"test.pattern.1"},
			TriggerType: moira.RisingTrigger,
		}

		trigger1Ver3 := &moira.Trigger{
			ID:          "triggerID-0000000000001",
			Name:        "test trigger 1 v3.0",
			Targets:     []string{"test.target.1"},
			Tags:        []string{"test-tag-2", "test-tag-3"},
			Patterns:    []string{"test.pattern.1"},
			TriggerType: moira.RisingTrigger,
		}

		subscription1Ver1 := &moira.SubscriptionData{
			ID:                "subscriptionID-00000000000001",
			Enabled:           true,
			Tags:              []string{"test-tag-1"},
			Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
			ThrottlingEnabled: true,
			User:              "user1",
		}

		subscription1Ver2 := &moira.SubscriptionData{
			ID:                "subscriptionID-00000000000001",
			Enabled:           true,
			Tags:              []string{"test-tag-2"},
			Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
			ThrottlingEnabled: true,
			User:              "user1",
		}

		Convey("Very simple trigger-subscription test", t, func(c C) {
			dataBase.flush()

			err := dataBase.SaveTrigger(trigger1Ver1.ID, trigger1Ver1)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err := dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldResemble, []string{trigger1Ver1.ID})

			err = dataBase.SaveSubscription(subscription1Ver1)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err = dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldBeEmpty)
		})

		Convey("Let's change trigger", t, func(c C) {
			// Add tags, don't remove old tags
			err := dataBase.SaveTrigger(trigger1Ver2.ID, trigger1Ver2)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err := dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldBeEmpty)

			// Remove old tag
			err = dataBase.SaveTrigger(trigger1Ver3.ID, trigger1Ver3)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err = dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldResemble, []string{trigger1Ver3.ID})
		})

		Convey("Let's change subscription", t, func(c C) {
			err := dataBase.SaveSubscription(subscription1Ver2)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err := dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldBeEmpty)
		})

		Convey("Mass operations with triggers and subscriptions", t, func(c C) {
			dataBase.flush()

			triggers := []*moira.Trigger{
				{
					ID:          "new-trigger-1",
					Name:        "Very New trigger 1",
					Targets:     []string{"new.target.1"},
					Tags:        []string{"new-tag-1"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
				{
					ID:          "new-trigger-2",
					Name:        "Very New trigger 2",
					Targets:     []string{"new.target.2"},
					Tags:        []string{"new-tag-2"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
				{
					ID:          "new-trigger-3",
					Name:        "Very New trigger 3",
					Targets:     []string{"new.target.3"},
					Tags:        []string{"new-tag-3"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
				{
					ID:          "new-trigger-4",
					Name:        "Very New trigger 4",
					Targets:     []string{"new.target.4"},
					Tags:        []string{"new-tag-1", "new-tag-2", "new-tag-3"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
				{
					ID:          "new-trigger-5",
					Name:        "Very New trigger 5",
					Targets:     []string{"new.target.5"},
					Tags:        []string{"new-tag-1", "new-tag-2"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
				{
					ID:          "new-trigger-6",
					Name:        "Very New trigger 6",
					Targets:     []string{"new.target.6"},
					Tags:        []string{"new-tag-5", "new-tag-6"},
					Patterns:    []string{"test.pattern.1"},
					TriggerType: moira.RisingTrigger,
				},
			}
			subscriptions := []*moira.SubscriptionData{
				{
					ID:                "new-subscriptionID-1",
					Enabled:           true,
					Tags:              []string{"new-tag-1"},
					Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
					ThrottlingEnabled: true,
					User:              "user1",
				},
				{
					ID:                "new-subscriptionID-2",
					Enabled:           true,
					Tags:              []string{"new-tag-2"},
					Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
					ThrottlingEnabled: true,
					User:              "user1",
				},
				{
					ID:                "new-subscriptionID-3",
					Enabled:           true,
					Tags:              []string{"new-tag-3"},
					Contacts:          []string{uuid.Must(uuid.NewV4()).String()},
					ThrottlingEnabled: true,
					User:              "user1",
				},
			}

			// Add new triggers
			for _, trigger := range triggers {
				err := dataBase.SaveTrigger(trigger.ID, trigger)
				c.So(err, ShouldBeNil)
			}

			unusedTriggerIDs, err := dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(len(unusedTriggerIDs), ShouldEqual, 6)

			// Add all subscriptions
			err = dataBase.SaveSubscriptions(subscriptions)
			c.So(err, ShouldBeNil)

			unusedTriggerIDs, err = dataBase.GetUnusedTriggerIDs()
			c.So(err, ShouldBeNil)
			c.So(unusedTriggerIDs, ShouldResemble, []string{triggers[5].ID})
		})
	})
}

func TestUnusedTriggersConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		err := dataBase.MarkTriggersAsUnused("123")
		c.So(err, ShouldNotBeNil)

		triggerIDs, err := dataBase.GetUnusedTriggerIDs()
		c.So(triggerIDs, ShouldBeEmpty)
		c.So(err, ShouldNotBeNil)
	})
}
