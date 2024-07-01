package main

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	goredis "github.com/go-redis/redis/v8"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUpdateTelegramUsersRecords(t *testing.T) {
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
			createOldTelegramUserRecords(database)

			Convey("When migration was applied", func() {
				err := updateTelegramUsersRecords(logger, database)
				So(err, ShouldBeNil)

				Convey("Database should be new", func() {
					result, err := client.Get(ctx, "moira-telegram-users:some telegram group").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, `{"chat_id":-1001494975744}`)

					result, err = client.Get(ctx, "moira-telegram-users:some telegram group failed migration").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, `{"chat_id":-1001494975755}`)

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, `{"chat_id":1}`)

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host:123").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "D4VdnzZDTS/xXF87THARWw==")
				})
			})
		})
	})
}

func TestDowngradeTelegramUsersRecords(t *testing.T) {
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

	Convey("Test data migration backwards", t, func() {
		Convey("Given new database", func() {
			createNewTelegramUserRecords(database)

			Convey("When migration was applied", func() {
				err := downgradeTelegramUsersRecords(logger, database)
				So(err, ShouldBeNil)

				Convey("Database should be old", func() {
					result, err := client.Get(ctx, "moira-telegram-users:some telegram group").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "-1001494975744")

					result, err = client.Get(ctx, "moira-telegram-users:some telegram group with topic").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, `{"chat_id":-1001494975766,"thread_id":1}`)

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "1")

					result, err = client.Get(ctx, "moira-telegram-users:@failed_migration").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "2")

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host:123").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "D4VdnzZDTS/xXF87THARWw==")
				})
			})
		})
	})
}

func createOldTelegramUserRecords(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.Set(ctx, "moira-telegram-users:some telegram group", "-1001494975744", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:some telegram group failed migration", `{"chat_id":-1001494975755}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@durov", "1", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:moira-bot-host:123", "D4VdnzZDTS/xXF87THARWw==", goredis.KeepTTL)
	}
}

func createNewTelegramUserRecords(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.Set(ctx, "moira-telegram-users:some telegram group", `{"chat_id":-1001494975744}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:some telegram group with topic", `{"chat_id":-1001494975766,"thread_id":1}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@durov", `{"chat_id":1}`, goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:@failed_migration", "2", goredis.KeepTTL)
		client.Set(ctx, "moira-telegram-users:moira-bot-host:123", "D4VdnzZDTS/xXF87THARWw==", goredis.KeepTTL)
	}
}
