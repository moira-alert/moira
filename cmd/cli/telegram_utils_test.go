package main

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

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
					So(result, ShouldEqual, "{\"chatId\":-1001494975744,\"type\":\"group\"}")

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "{\"chatId\":1,\"type\":\"private\"}")

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host").Result()
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

					result, err = client.Get(ctx, "moira-telegram-users:@durov").Result()
					So(err, ShouldBeNil)
					So(result, ShouldEqual, "1")

					result, err = client.Get(ctx, "moira-telegram-users:moira-bot-host").Result()
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

		client.Set(ctx, "moira-telegram-users:some telegram group", "-1001494975744", -1)
		client.Set(ctx, "moira-telegram-users:@durov", "1", -1)
		client.Set(ctx, "moira-telegram-users:moira-bot-host", "D4VdnzZDTS/xXF87THARWw==", -1)
	}
}

func createNewTelegramUserRecords(database moira.Database) {
	switch d := database.(type) {
	case *redis.DbConnector:
		d.Flush()
		client := d.Client()
		ctx := d.Context()

		client.Set(ctx, "moira-telegram-users:some telegram group", "{\"type\":\"group\",\"chatId\":-1001494975744}", -1)
		client.Set(ctx, "moira-telegram-users:@durov", "{\"type\":\"private\",\"chatId\":1}", -1)
		client.Set(ctx, "moira-telegram-users:moira-bot-host", "D4VdnzZDTS/xXF87THARWw==", -1)
	}
}
