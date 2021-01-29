package redis

import (
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

var logger, _ = logging.ConfigureLog("stdout", "debug", "test", true)

var sentinelConfig = SentinelPoolDialerConfig{
	MasterName:  "master01", //nolint
	DB:          0,
	DialTimeout: time.Millisecond,
	SentinelAddresses: []string{
		"fake-sentinel:26379",
	},
}

func TestDirectDialer(t *testing.T) {
	Convey("Direct dialer", t, func() {
		Convey("Tries dial and fails", func() {
			dialer := DirectPoolDialer{ //nolint
				serverAddress: "fake-redis:6379",
				db:            0, //nolint
				dialTimeout:   time.Millisecond,
			}

			_, err := dialer.Dial()

			So(err.Error(), ShouldEqual, "dial tcp: i/o timeout")
		})
		//nolint
		Convey("Test dials successfully", func() {
			dialer := DirectPoolDialer{ //nolint
				serverAddress: "localhost:6379",
				db:            0, //nolint
				dialTimeout:   5 * time.Second,
			}

			conn, err := dialer.Dial()

			So(err, ShouldBeNil)
			//nolint
			err = dialer.Test(conn, time.Now())
			So(err, ShouldBeNil)
		})
	})
}

func TestSentinelDialer(t *testing.T) {
	dialer := NewSentinelPoolDialer(logger, sentinelConfig)

	Convey("Tries dial and fails", t, func() {
		_, err := dialer.Dial()
		So(err.Error(), ShouldEqual, "redigo: no sentinels available; last error: dial tcp: i/o timeout")
	})
}

func TestSlaveDialer(t *testing.T) { //nolint
	dialer := NewSentinelPoolDialer(logger, sentinelConfig)
	slaveDialer := NewSentinelSlavePoolDialer(dialer)

	Convey("Tries dial and fails", t, func() {
		_, err := slaveDialer.Dial()
		So(err.Error(), ShouldEqual, "redigo: no sentinels available; last error: dial tcp: i/o timeout")
	})
} //nolint
