package redis

import (
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

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
