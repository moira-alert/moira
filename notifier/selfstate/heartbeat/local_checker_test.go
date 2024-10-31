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
	defaultLocalCheckDelay = time.Minute
)

func TestNewLocalCheckerHeartbeater(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	validationErr := validator.ValidationErrors{}

	Convey("Test NewLocalCheckerHeartbeater", t, func() {
		Convey("With too low local check delay", func() {
			cfg := LocalCheckerHeartbeaterConfig{
				LocalCheckDelay: -1,
			}

			localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(localCheckerHeartbeater, ShouldBeNil)
		})

		Convey("Without local check delay", func() {
			cfg := LocalCheckerHeartbeaterConfig{}

			localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(localCheckerHeartbeater, ShouldBeNil)
		})

		Convey("With correct local checker heartbeater config", func() {
			cfg := LocalCheckerHeartbeaterConfig{
				LocalCheckDelay: 1,
			}

			expected := &localCheckerHeartbeater{
				heartbeaterBase: heartbeaterBase,
				cfg:             cfg,
			}

			localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
			So(err, ShouldBeNil)
			So(localCheckerHeartbeater, ShouldResemble, expected)
		})
	})
}

func TestLocalCheckerHeartbeaterCheck(t *testing.T) {
	database, clock, testTime, heartbeaterBase := heartbeaterHelper(t)

	cfg := LocalCheckerHeartbeaterConfig{
		LocalCheckDelay: defaultMetricReceivedDelay,
	}

	localCheckerHeartbeater, _ := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)

	var (
		testErr                                        = errors.New("test error")
		triggersToCheckCount, checksUpdatesCount int64 = 10, 10
	)

	Convey("Test localCheckerHeartbeater.Check", t, func() {
		Convey("With GetTriggersToCheckCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, testErr)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With GetChecksUpdatesCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetChecksUpdatesCount().Return(checksUpdatesCount, testErr)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With last checks count not equal current checks count", func() {
			defer func() {
				localCheckerHeartbeater.lastChecksCount = 0
			}()

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetChecksUpdatesCount().Return(checksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(localCheckerHeartbeater.lastChecksCount, ShouldResemble, checksUpdatesCount)
		})

		Convey("With zero triggers to check count", func() {
			defer func() {
				localCheckerHeartbeater.lastChecksCount = 0
			}()

			var zeroTriggersToCheckCount int64

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(zeroTriggersToCheckCount, nil)
			database.EXPECT().GetChecksUpdatesCount().Return(checksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(localCheckerHeartbeater.lastChecksCount, ShouldResemble, checksUpdatesCount)
		})

		localCheckerHeartbeater.lastChecksCount = checksUpdatesCount

		Convey("With too much time elapsed since the last successful check", func() {
			localCheckerHeartbeater.lastSuccessfulCheck = testTime.Add(-10 * defaultLocalCheckDelay)
			defer func() {
				localCheckerHeartbeater.lastSuccessfulCheck = testTime
			}()

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetChecksUpdatesCount().Return(checksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateError)
		})

		Convey("With short time elapsed since the last successful check", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetChecksUpdatesCount().Return(checksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := localCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
		})
	})
}

func TestLocalCheckerHeartbeaterNeedTurnOffNotifier(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test localCheckerHeartbeater.TurnOffNotifier", t, func() {
		cfg := LocalCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedTurnOffNotifier: true,
			},
			LocalCheckDelay: defaultLocalCheckDelay,
		}

		localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needTurnOffNotifier := localCheckerHeartbeater.NeedTurnOffNotifier()
		So(needTurnOffNotifier, ShouldBeTrue)
	})
}

func TestLocalCheckerHeartbeaterType(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test localCheckerHeartbeater.Type", t, func() {
		cfg := LocalCheckerHeartbeaterConfig{
			LocalCheckDelay: defaultLocalCheckDelay,
		}

		localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		localCheckerHeartbeaterType := localCheckerHeartbeater.Type()
		So(localCheckerHeartbeaterType, ShouldResemble, datatypes.HearbeatTypeNotSet)
	})
}

func TestLocalCheckerHeartbeaterAlertSettings(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test localCheckerHeartbeater.AlertSettings", t, func() {
		alertCfg := AlertConfig{
			Name: "test name",
			Desc: "test desc",
		}

		cfg := LocalCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				AlertCfg: alertCfg,
			},
			LocalCheckDelay: defaultLocalCheckDelay,
		}

		localCheckerHeartbeater, err := NewLocalCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		alertSettings := localCheckerHeartbeater.AlertSettings()
		So(alertSettings, ShouldResemble, alertCfg)
	})
}
