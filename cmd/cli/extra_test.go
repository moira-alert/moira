package main

import (
	"testing"

	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMakeAllTriggersDataSourceLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	conf := getDefault()
	logger, err := logging.ConfigureLog(conf.LogFile, "error", "cli", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}

	database := redis.NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()
	client := database.Client()
	ctx := database.Context()

	Convey("Test makeAllTriggersDataSourceLocal", t, func() {
		Convey("Given some different triggers (local and remote)", func() {
			localTrigger1 := moira.Trigger{
				ID:       "triggerID-0000000000001",
				Name:     "test trigger 1",
				IsRemote: false,
			}
			_ = database.SaveTrigger(localTrigger1.ID, &localTrigger1)
			trigger, _ := database.GetTrigger(localTrigger1.ID)
			So(trigger.IsRemote, ShouldBeFalse)

			localTrigger2 := moira.Trigger{
				ID:       "triggerID-0000000000002",
				Name:     "test trigger 2",
				IsRemote: false,
			}
			_ = database.SaveTrigger(localTrigger2.ID, &localTrigger2)
			trigger, _ = database.GetTrigger(localTrigger2.ID)
			So(trigger.IsRemote, ShouldBeFalse)

			remoteTrigger1 := moira.Trigger{
				ID:       "triggerID-0000000000003",
				Name:     "test trigger 3",
				IsRemote: true,
			}
			_ = database.SaveTrigger(remoteTrigger1.ID, &remoteTrigger1)
			trigger, _ = database.GetTrigger(remoteTrigger1.ID)
			So(trigger.IsRemote, ShouldBeTrue)

			remoteTrigger2 := moira.Trigger{
				ID:       "triggerID-0000000000004",
				Name:     "test trigger 4",
				IsRemote: true,
			}
			_ = database.SaveTrigger(remoteTrigger2.ID, &remoteTrigger2)
			trigger, _ = database.GetTrigger(remoteTrigger2.ID)
			So(trigger.IsRemote, ShouldBeTrue)

			valueStoredAtKey, _ := client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Result()
			So(valueStoredAtKey, ShouldHaveLength, 2)
			valueStoredAtKey, _ = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Result()
			So(valueStoredAtKey, ShouldHaveLength, 4)

			Convey("When makeAllTriggersDataSourceLocal was called", func() {
				err := makeAllTriggersDataSourceLocal(logger, database)
				So(err, ShouldBeNil)

				Convey("All triggers should be local", func() {
					trigger, _ := database.GetTrigger(localTrigger1.ID)
					So(trigger.IsRemote, ShouldBeFalse)

					trigger, _ = database.GetTrigger(localTrigger2.ID)
					So(trigger.IsRemote, ShouldBeFalse)

					trigger, _ = database.GetTrigger(remoteTrigger1.ID)
					So(trigger.IsRemote, ShouldBeFalse)

					trigger, _ = database.GetTrigger(remoteTrigger1.ID)
					So(trigger.IsRemote, ShouldBeFalse)

					valueStoredAtKey, _ := client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Result()
					So(valueStoredAtKey, ShouldHaveLength, 0)
					valueStoredAtKey, _ = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Result()
					So(valueStoredAtKey, ShouldHaveLength, 4)
				})
			})
		})
	})
}
