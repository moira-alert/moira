package redis

import (
	"os"
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

var redisPort = os.Getenv("REDIS_PORT")
var redisHost = os.Getenv("REDIS_HOST")

var config = Config{Port: redisPort, Host: redisHost}
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
