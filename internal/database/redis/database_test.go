package redis

import (
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

var config = Config{Port: "6379", Host: "0.0.0.0"}
var emptyConfig = Config{}
var testSource = DBSource("test")

// use it only for tests
func newTestDatabase(logger moira2.Logger, config Config) *DbConnector {
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
