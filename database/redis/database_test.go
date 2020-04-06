package redis

import (
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

var config = Config{Port: "6379", Host: "0.0.0.0"}
var emptyConfig = Config{}
var testSource = DBSource("test")

// docker run -p 6379:6379 redis
func TestInitialization(t *testing.T) {
	Convey("Initialization methods", t, func() {
		logger, _ := logging.ConfigureLog("stdout", "info", "test")
		database := NewTestDatabase(logger, emptyConfig,testSource)
		So(database, ShouldNotBeEmpty)
		_, err := database.pool.Dial()
		So(err, ShouldNotBeNil)
	})
}
