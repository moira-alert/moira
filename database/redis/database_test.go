package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewDatabase(t *testing.T) {
	Convey("NewDatabase should return correct DBConnector", t, func() {
		logger, _ := logging.ConfigureLog("stdout", "info", "test", true) // nolint: govet
		database := NewTestDatabase(logger)
		So(database, ShouldNotBeEmpty)
		So(database.source, ShouldEqual, "test")
		So(database.logger, ShouldEqual, logger)
		So(database.context, ShouldResemble, context.Background())

		database.Flush()
		defer database.Flush()

		Convey("Redis client must be workable", func() {
			ctx := context.Background()

			Convey("Can get the value of key that does not exists", func() {
				err := (*database.client).Get(ctx, "key").Err()
				So(err, ShouldEqual, redis.Nil)
			})

			Convey("Can set key to hold the string value", func() {
				err := (*database.client).Set(ctx, "key", "value", 0).Err()
				So(err, ShouldBeNil)
			})

			Convey("Can get the value of key that exists", func() {
				(*database.client).Set(ctx, "key", "value", 0)

				val, err := (*database.client).Get(ctx, "key").Result()
				So(err, ShouldBeNil)
				So(val, ShouldEqual, "value")
			})

			Convey("Can remove key", func() {
				(*database.client).Set(ctx, "key", "value", 0)
				val := (*database.client).Get(ctx, "key").Val()
				So(val, ShouldEqual, "value")

				err := (*database.client).Del(ctx, "key").Err()
				So(err, ShouldBeNil)

				err = (*database.client).Get(ctx, "key").Err()
				So(err, ShouldEqual, redis.Nil)
			})
		})
	})
}
