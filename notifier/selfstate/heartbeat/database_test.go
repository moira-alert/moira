package heartbeat

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira/datatypes"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	defaultRedisDisconnectDelay = time.Minute
)

func TestNewDatabaseHeartbeater(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	validationErr := validator.ValidationErrors{}

	Convey("Test NewDatabaseHeartbeater", t, func() {
		Convey("With too low redis disconnect delay", func() {
			cfg := DatabaseHeartbeaterConfig{
				HeartbeaterBaseConfig: HeartbeaterBaseConfig{
					Enabled: true,
				},
				RedisDisconnectDelay: -1,
			}

			databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(databaseHeartbeater, ShouldBeNil)
		})

		Convey("Without redis disconnect delay", func() {
			cfg := DatabaseHeartbeaterConfig{}

			databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(databaseHeartbeater, ShouldBeNil)
		})

		Convey("With correct database heartbeater config", func() {
			cfg := DatabaseHeartbeaterConfig{
				RedisDisconnectDelay: 1,
			}

			expected := &databaseHeartbeater{
				heartbeaterBase: heartbeaterBase,
				cfg:             cfg,
			}

			databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
			So(err, ShouldBeNil)
			So(databaseHeartbeater, ShouldResemble, expected)
		})
	})
}

func TestDatabaseHeartbeaterCheck(t *testing.T) {
	database, clock, testTime, heartbeaterBase := heartbeaterHelper(t)

	cfg := DatabaseHeartbeaterConfig{
		RedisDisconnectDelay: defaultRedisDisconnectDelay,
	}

	databaseHeartbeater, _ := NewDatabaseHeartbeater(cfg, heartbeaterBase)

	var (
		testErr      = errors.New("test error")
		checkUpdates int64
	)

	Convey("Test databaseHeartbeater.Check", t, func() {
		Convey("With nil error in GetCheckUpdatedCount", func() {
			database.EXPECT().GetChecksUpdatesCount().Return(checkUpdates, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := databaseHeartbeater.Check()
			So(state, ShouldResemble, StateOK)
			So(err, ShouldBeNil)
		})

		Convey("With too much time elapsed since the last successful check", func() {
			heartbeaterBase.lastSuccessfulCheck = testTime.Add(-10 * defaultRedisDisconnectDelay)
			defer func() {
				heartbeaterBase.lastSuccessfulCheck = testTime
			}()

			database.EXPECT().GetChecksUpdatesCount().Return(checkUpdates, testErr)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := databaseHeartbeater.Check()
			So(state, ShouldResemble, StateError)
			So(err, ShouldBeNil)
		})

		Convey("With only error from GetChecksUpdateCount", func() {
			database.EXPECT().GetChecksUpdatesCount().Return(checkUpdates, testErr)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := databaseHeartbeater.Check()
			So(state, ShouldResemble, StateOK)
			So(err, ShouldResemble, testErr)
		})
	})
}

func TestDatabaseHeartbeaterNeedTurnOffNotifier(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test databaseHeartbeater.TurnOffNotifier", t, func() {
		cfg := DatabaseHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedTurnOffNotifier: true,
			},
			RedisDisconnectDelay: defaultRedisDisconnectDelay,
		}

		databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needTurnOffNotifier := databaseHeartbeater.NeedTurnOffNotifier()
		So(needTurnOffNotifier, ShouldBeTrue)
	})
}

func TestDatabaseHeartbeaterType(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test databaseHeartbeater.Type", t, func() {
		cfg := DatabaseHeartbeaterConfig{
			RedisDisconnectDelay: defaultRedisDisconnectDelay,
		}

		databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		databaseHeartbeaterType := databaseHeartbeater.Type()
		So(databaseHeartbeaterType, ShouldResemble, datatypes.HeartbeatTypeNotSet)
	})
}

func TestDatabaseHeartbeaterAlertSettings(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test databaseHeartbeater.AlertSettings", t, func() {
		alertCfg := AlertConfig{
			Name: "test name",
			Desc: "test desc",
		}

		cfg := DatabaseHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				AlertCfg: alertCfg,
			},
			RedisDisconnectDelay: defaultRedisDisconnectDelay,
		}

		databaseHeartbeater, err := NewDatabaseHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		alertSettings := databaseHeartbeater.AlertSettings()
		So(alertSettings, ShouldResemble, alertCfg)
	})
}
