package redis

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/logging/go-logging"
)

func TestRenewNodataCheckerRegistration(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	lockTTLMilliseconds := 3000
	lockTTLDuration := time.Duration(lockTTLMilliseconds) * time.Millisecond

	Convey("Manage NODATA checker registrations", t, func() {
		Convey("No registrations to renew", func() {
			renewed := dataBase.RenewNodataCheckerRegistration()
			So(renewed, ShouldBeFalse)
		})
		Convey("Just register, should be registered", func() {
			registered := dataBase.RegisterNodataCheckerIfAlreadyNot(lockTTLDuration)
			So(registered, ShouldBeTrue)
		})
		Convey("Renew NODATA checker registration, should be renewed", func() {
			renewed := dataBase.RenewNodataCheckerRegistration()
			So(renewed, ShouldBeTrue)
		})
		Convey("Renew bot registration, should not be renewed", func() {
			lockResults := testLockWithTTLExpireErrorExpected(lockTTLMilliseconds, 2, func() bool {
				return dataBase.RenewNodataCheckerRegistration()
			})
			So(lockResults[len(lockResults)-1], ShouldBeFalse)
		})
	})
}
