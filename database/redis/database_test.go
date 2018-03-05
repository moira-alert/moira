package redis

import (
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInitialization(t *testing.T) {
	Convey("Initialization methods", t, func() {
		logger, _ := logging.ConfigureLog("stdout", "info", "test")
		database := NewDatabase(logger, emptyConfig)
		So(database, ShouldNotBeEmpty)
		_, err := database.pool.Dial()
		So(err, ShouldNotBeNil)
	})
}
