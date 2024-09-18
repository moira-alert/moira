package main

import (
	"testing"

	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	"github.com/moira-alert/moira"

	rds "github.com/go-redis/redis/v8"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCluster(t *testing.T) {
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

	Convey("Test data migration forwards", t, func() {
		Convey("Given old database", func() {
			createDataWithOldKeys(database)

			valueStoredAtKey := client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
			So(valueStoredAtKey, ShouldBeEmpty)

			valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
			So(valueStoredAtKey, ShouldBeEmpty)

			valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
			So(valueStoredAtKey, ShouldBeEmpty)

			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
			So(valueStoredAtKey, ShouldBeEmpty)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
			So(valueStoredAtKey, ShouldBeEmpty)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
			So(valueStoredAtKey, ShouldBeEmpty)

			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
			So(valueStoredAtKey, ShouldBeEmpty)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
			So(valueStoredAtKey, ShouldBeEmpty)
			valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
			So(len(valueStoredAtKey), ShouldResemble, 3)
			valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
			So(valueStoredAtKey, ShouldBeEmpty)

			Convey("When migration was applied", func() {
				err := addRedisClusterSupport(logger, database)
				So(err, ShouldBeNil)

				Convey("Database should be new", func() {
					valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)

					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)
					valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
					So(valueStoredAtKey, ShouldBeEmpty)
					valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
					So(len(valueStoredAtKey), ShouldResemble, 3)
				})
			})
		})

		Convey("Test data migration reverse", func() {
			Convey("Given new database", func() {
				createDataWithNewKeys(database)

				valueStoredAtKey := client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
				So(valueStoredAtKey, ShouldBeEmpty)

				valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
				So(valueStoredAtKey, ShouldBeEmpty)

				valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
				So(valueStoredAtKey, ShouldBeEmpty)

				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
				So(valueStoredAtKey, ShouldBeEmpty)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
				So(valueStoredAtKey, ShouldBeEmpty)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
				So(valueStoredAtKey, ShouldBeEmpty)

				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
				So(valueStoredAtKey, ShouldBeEmpty)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
				So(valueStoredAtKey, ShouldBeEmpty)
				valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
				So(len(valueStoredAtKey), ShouldResemble, 3)
				valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
				So(valueStoredAtKey, ShouldBeEmpty)

				Convey("When migration was reversed", func() {
					err := removeRedisClusterSupport(logger, database)
					So(err, ShouldBeNil)

					Convey("Database should be old", func() {
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-any-tags-subscriptions").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-triggers-list").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-triggers-list").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-triggers-list}:moira-remote-triggers-list").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-remote-triggers-list").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag1").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag1").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag2").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag2").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-subscriptions}:tag3").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-subscriptions:tag3").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)

						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag1").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag1").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag2").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag2").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)
						valueStoredAtKey = client.SMembers(ctx, "{moira-tag-triggers}:tag3").Val()
						So(valueStoredAtKey, ShouldBeEmpty)
						valueStoredAtKey = client.SMembers(ctx, "moira-tag-triggers:tag3").Val()
						So(len(valueStoredAtKey), ShouldResemble, 3)
					})
				})
			})
		})
	})
}

func createDataWithOldKeys(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-any-tags-subscriptions", "subscriptionID-00000000000003")

		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-triggers-list", "triggerID-0000000000003")

		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000004")
		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000005")
		client.SAdd(ctx, "moira-remote-triggers-list", "triggerID-0000000000006")

		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag1", "subscriptionID-00000000000003")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag2", "subscriptionID-00000000000003")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000001")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000002")
		client.SAdd(ctx, "moira-tag-subscriptions:tag3", "subscriptionID-00000000000003")

		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag1", "triggerID-0000000000003")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag2", "triggerID-0000000000003")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000001")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000002")
		client.SAdd(ctx, "moira-tag-triggers:tag3", "triggerID-0000000000003")
	}
}

func createDataWithNewKeys(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:moira-any-tags-subscriptions", "subscriptionID-00000000000003")

		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-triggers-list}:moira-triggers-list", "triggerID-0000000000003")

		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000004")
		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000005")
		client.SAdd(ctx, "{moira-triggers-list}:moira-remote-triggers-list", "triggerID-0000000000006")

		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag1", "subscriptionID-00000000000003")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag2", "subscriptionID-00000000000003")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000001")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000002")
		client.SAdd(ctx, "{moira-tag-subscriptions}:tag3", "subscriptionID-00000000000003")

		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag1", "triggerID-0000000000003")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag2", "triggerID-0000000000003")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000001")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000002")
		client.SAdd(ctx, "{moira-tag-triggers}:tag3", "triggerID-0000000000003")
	}
}

func Test_renameKey(t *testing.T) {
	logger, _ := logging.GetLogger("Test Worker")
	oldKey := "my_test_key"
	newKey := "my_new_test_key"

	Convey("Something was renamed", t, func() {
		database := redis.NewTestDatabase(logger)
		database.Flush()
		defer database.Flush()
		err := database.Client().Set(database.Context(), oldKey, "123", 0).Err()
		So(err, ShouldBeNil)

		err = renameKey(database, oldKey, newKey)
		So(err, ShouldBeNil)

		res, err := database.Client().Get(database.Context(), newKey).Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "123")
		err = database.Client().Get(database.Context(), oldKey).Err()
		So(err, ShouldEqual, rds.Nil)
	})

	Convey("Nothing was renamed", t, func() {
		database := redis.NewTestDatabase(logger)
		database.Flush()
		defer database.Flush()
		err := database.Client().Set(database.Context(), oldKey, "123", 0).Err()
		So(err, ShouldBeNil)

		err = renameKey(database, "no_exist_key", newKey)
		So(err, ShouldBeNil)

		err = database.Client().Get(database.Context(), newKey).Err()
		So(err, ShouldEqual, rds.Nil)

		res, err := database.Client().Get(database.Context(), oldKey).Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "123")
	})
}

func Test_changeKeysPrefix(t *testing.T) {
	logger, _ := logging.GetLogger("Test Worker")
	oldKey := "my_test_key"
	newKey := "my_new_test_key"

	Convey("Something was renamed", t, func() {
		database := redis.NewTestDatabase(logger)
		database.Flush()
		defer database.Flush()
		err := database.Client().Set(database.Context(), oldKey+"1", "1", 0).Err()
		So(err, ShouldBeNil)
		err = database.Client().Set(database.Context(), oldKey+"2", "2", 0).Err()
		So(err, ShouldBeNil)
		err = database.Client().Set(database.Context(), oldKey+"3", "3", 0).Err()
		So(err, ShouldBeNil)

		err = changeKeysPrefix(database, oldKey, newKey)
		So(err, ShouldBeNil)

		res, err := database.Client().Get(database.Context(), newKey+"1").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "1")
		res, err = database.Client().Get(database.Context(), newKey+"2").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "2")
		res, err = database.Client().Get(database.Context(), newKey+"3").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "3")
		err = database.Client().Get(database.Context(), oldKey+"1").Err()
		So(err, ShouldEqual, rds.Nil)
		err = database.Client().Get(database.Context(), oldKey+"2").Err()
		So(err, ShouldEqual, rds.Nil)
		err = database.Client().Get(database.Context(), oldKey+"3").Err()
		So(err, ShouldEqual, rds.Nil)
	})

	Convey("Nothing was renamed", t, func() {
		database := redis.NewTestDatabase(logger)
		database.Flush()
		defer database.Flush()
		err := database.Client().Set(database.Context(), oldKey+"1", "1", 0).Err()
		So(err, ShouldBeNil)
		err = database.Client().Set(database.Context(), oldKey+"2", "2", 0).Err()
		So(err, ShouldBeNil)
		err = database.Client().Set(database.Context(), oldKey+"3", "3", 0).Err()
		So(err, ShouldBeNil)

		err = renameKey(database, "no_exist_key", newKey)
		So(err, ShouldBeNil)

		err = database.Client().Get(database.Context(), newKey).Err()
		So(err, ShouldEqual, rds.Nil)

		res, err := database.Client().Get(database.Context(), oldKey+"1").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "1")
		res, err = database.Client().Get(database.Context(), oldKey+"2").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "2")
		res, err = database.Client().Get(database.Context(), oldKey+"3").Result()
		So(err, ShouldBeNil)
		So(res, ShouldResemble, "3")
	})
}
