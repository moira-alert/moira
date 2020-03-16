package redis

import (
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

var config = Config{Port: "6379", Host: "0.0.0.0"}
var emptyConfig = Config{}
var testSource = DBSource("test")

// use it only for tests
func newTestDatabase(logger moira.Logger, config Config) *DbConnector {
	return NewDatabase(logger, config, testSource)
}

// docker run -p 6379:6379 redis
func TestInitialization(t *testing.T) {
	Convey("Initialization methods", t, func() {
		logger, _ := logging.ConfigureLog("stdout", "info", "test")
		database := newTestDatabase(logger, emptyConfig)
		So(database, ShouldNotBeEmpty)
		_, err := database.pool.Dial()
		So(err, ShouldNotBeNil)
	})
}

func TestAllowStale(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")

	Convey("Allow stale", t, func() {
		Convey("When using redis directly, returns same db", func() {
			database := newTestDatabase(logger, emptyConfig)

			staleDatabase := database.AllowStale()

			So(staleDatabase, ShouldPointTo, database)
		})

		Convey("When using sentinel, returns slave db instance, retains references", func() {
			sentinelConfig := Config{
				MasterName:        "mstr",
				SentinelAddresses: []string{"addr.ru"},
				DB:                0,
				AllowSlaveReads:   true,
			}
			database := newTestDatabase(logger, sentinelConfig)

			staleDatabase := database.AllowStale()

			So(staleDatabase, ShouldNotPointTo, database)
			staleConnector := staleDatabase.(*DbConnector)
			So(staleConnector.metricsCache, ShouldPointTo, database.metricsCache)
			So(staleConnector.retentionCache, ShouldPointTo, database.retentionCache)
			So(staleConnector.retentionSavingCache, ShouldPointTo, database.retentionSavingCache)
			So(staleConnector.sync, ShouldPointTo, database.sync)
		})
	})
}
