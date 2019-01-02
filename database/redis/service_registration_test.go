package redis

import (
	"github.com/moira-alert/moira"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/logging/go-logging"
)

func TestRenewServiceRegistration(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	lockTTLMilliseconds := 3000
	lockTTLDuration := time.Duration(lockTTLMilliseconds) * time.Millisecond

	Convey("Manage service registrations", t, func() {
		Convey("No registrations to renew", func() {
			renewed := dataBase.RenewServiceRegistration(service1Registration)
			So(renewed, ShouldBeFalse)
		})
		Convey("Just register, should be registered", func() {
			registered := dataBase.RegisterServiceIfNotDone(service1Registration, lockTTLDuration)
			So(registered, ShouldBeTrue)
		})
		Convey("Renew service registration, should be renewed", func() {
			renewed := dataBase.RenewServiceRegistration(service1Registration)
			So(renewed, ShouldBeTrue)
		})
		Convey("Renew service registration, should not be renewed", func() {
			lockResults := testLockWithTTLExpireErrorExpected(lockTTLMilliseconds, 2, func() bool {
				return dataBase.RenewServiceRegistration(service1Registration)
			})
			So(lockResults[len(lockResults)-1], ShouldBeFalse)
		})
		Convey("Deregister service", func() {
			So(dataBase.DeregisterService(service1Registration), ShouldBeTrue)
		})

		Convey("Check several registration at once", func() {
			Convey("Register 3 services at once", func() {
				So(dataBase.RegisterServiceIfNotDone(service1Registration, lockTTLDuration), ShouldBeTrue)
				So(dataBase.RegisterServiceIfNotDone(service2Registration, lockTTLDuration), ShouldBeTrue)
				So(dataBase.RegisterServiceIfNotDone(service3Registration, lockTTLDuration), ShouldBeTrue)
			})
			Convey("Renew 3 services at once", func() {
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeTrue)
			})
			Convey("Deregister 3 services at once", func() {
				So(dataBase.DeregisterService(service1Registration), ShouldBeTrue)
				So(dataBase.DeregisterService(service2Registration), ShouldBeTrue)
				So(dataBase.DeregisterService(service3Registration), ShouldBeTrue)
			})
			Convey("Register 1 service, check all 3", func() {
				So(dataBase.RegisterServiceIfNotDone(service1Registration, lockTTLDuration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeFalse)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeFalse)
			})
			Convey("Register 1 more service, check all 3", func() {
				So(dataBase.RegisterServiceIfNotDone(service2Registration, lockTTLDuration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeFalse)
			})
			Convey("Register the last service, check all 3", func() {
				So(dataBase.RegisterServiceIfNotDone(service3Registration, lockTTLDuration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeTrue)
			})
			Convey("Deregister 1 service, check all 3", func() {
				So(dataBase.DeregisterService(service1Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeFalse)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeTrue)
			})
			Convey("Deregister 2 other, check all 3", func() {
				So(dataBase.DeregisterService(service1Registration), ShouldBeTrue)
				So(dataBase.DeregisterService(service2Registration), ShouldBeTrue)
				So(dataBase.DeregisterService(service3Registration), ShouldBeTrue)
				So(dataBase.RenewServiceRegistration(service1Registration), ShouldBeFalse)
				So(dataBase.RenewServiceRegistration(service2Registration), ShouldBeFalse)
				So(dataBase.RenewServiceRegistration(service3Registration), ShouldBeFalse)
			})
		})

	})
}

var service1Registration moira.SingleInstanceService = "test:service1"
var service2Registration moira.SingleInstanceService = "test:service2"
var service3Registration moira.SingleInstanceService = "test:service3"
