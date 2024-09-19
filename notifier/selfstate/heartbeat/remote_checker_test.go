package heartbeat

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	defaultRemoteCheckDelay = time.Minute
)

func TestNewRemoteCheckerHeartbeater(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	validationErr := validator.ValidationErrors{}

	Convey("Test NewRemoteCheckerHeartbeater", t, func() {
		Convey("With too low remote check delay", func() {
			cfg := RemoteCheckerHeartbeaterConfig{
				RemoteCheckDelay: -1,
			}

			remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(remoteCheckerHeartbeater, ShouldBeNil)
		})

		Convey("Without remote check delay", func() {
			cfg := RemoteCheckerHeartbeaterConfig{}

			remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(remoteCheckerHeartbeater, ShouldBeNil)
		})

		Convey("With correct remote checker heartbeater config", func() {
			cfg := RemoteCheckerHeartbeaterConfig{
				RemoteCheckDelay: 1,
			}

			expected := &remoteCheckerHeartbeater{
				heartbeaterBase: heartbeaterBase,
				cfg:             cfg,
			}

			remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
			So(err, ShouldBeNil)
			So(remoteCheckerHeartbeater, ShouldResemble, expected)
		})
	})
}

func TestRemoteCheckerHeartbeaterCheck(t *testing.T) {
	database, clock, testTime, heartbeaterBase := heartbeaterHelper(t)

	cfg := RemoteCheckerHeartbeaterConfig{
		RemoteCheckDelay: defaultRemoteCheckDelay,
	}

	remoteCheckerHeartbeater, _ := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)

	var (
		testErr                                              = errors.New("test error")
		triggersToCheckCount, remoteChecksUpdatesCount int64 = 10, 10
	)

	Convey("Test remoteCheckerHeartbeater.Check", t, func() {
		Convey("With GetTriggersToCheckCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(triggersToCheckCount, testErr)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With GetRemoteChecksUpdatesCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(remoteChecksUpdatesCount, testErr)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With last remote checks count not equal current remote checks count", func() {
			defer func() {
				remoteCheckerHeartbeater.lastRemoteChecksCount = 0
			}()

			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(remoteChecksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(remoteCheckerHeartbeater.lastRemoteChecksCount, ShouldResemble, remoteChecksUpdatesCount)
		})

		Convey("With zero triggers to check count", func() {
			defer func() {
				remoteCheckerHeartbeater.lastRemoteChecksCount = 0
			}()

			var zeroTriggersToCheckCount int64

			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(zeroTriggersToCheckCount, nil)
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(remoteChecksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(remoteCheckerHeartbeater.lastRemoteChecksCount, ShouldResemble, remoteChecksUpdatesCount)
		})

		remoteCheckerHeartbeater.lastRemoteChecksCount = remoteChecksUpdatesCount

		Convey("With too much time elapsed since the last successful check", func() {
			remoteCheckerHeartbeater.lastSuccessfulCheck = testTime.Add(-10 * defaultRemoteCheckDelay)
			defer func() {
				remoteCheckerHeartbeater.lastSuccessfulCheck = testTime
			}()

			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(remoteChecksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateError)
		})

		Convey("With short time elapsed since the last successful check", func() {
			database.EXPECT().GetTriggersToCheckCount(remoteClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(remoteChecksUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := remoteCheckerHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
		})
	})
}

func TestRemoteCheckerHeartbeaterNeedTurnOffNotifier(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test remoteCheckerHeartbeater.TurnOffNotifier", t, func() {
		cfg := RemoteCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedTurnOffNotifier: true,
			},
			RemoteCheckDelay: defaultRemoteCheckDelay,
		}

		remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needTurnOffNotifier := remoteCheckerHeartbeater.NeedTurnOffNotifier()
		So(needTurnOffNotifier, ShouldBeTrue)
	})
}

func TestRemoteCheckerHeartbeaterNeedToCheckOthers(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test remoteCheckerHeartbeater.NeedToCheckOthers", t, func() {
		cfg := RemoteCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedToCheckOthers: true,
			},
			RemoteCheckDelay: defaultRemoteCheckDelay,
		}

		remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needToCheckOthers := remoteCheckerHeartbeater.NeedToCheckOthers()
		So(needToCheckOthers, ShouldBeTrue)
	})
}

func TestRemoteCheckerHeartbeaterType(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test remoteCheckerHeartbeater.Type", t, func() {
		cfg := RemoteCheckerHeartbeaterConfig{
			RemoteCheckDelay: defaultRemoteCheckDelay,
		}

		remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		remoteCheckerHeartbeaterType := remoteCheckerHeartbeater.Type()
		So(remoteCheckerHeartbeaterType, ShouldResemble, moira.EmergencyTypeRemoteCheckerNoTriggerCheck)
	})
}

func TestRemoteCheckerHeartbeaterAlertSettings(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test remoteCheckerHeartbeater.AlertSettings", t, func() {
		alertCfg := AlertConfig{
			Name: "test name",
			Desc: "test desc",
		}

		cfg := RemoteCheckerHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				AlertCfg: alertCfg,
			},
			RemoteCheckDelay: defaultRemoteCheckDelay,
		}

		remoteCheckerHeartbeater, err := NewRemoteCheckerHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		alertSettings := remoteCheckerHeartbeater.AlertSettings()
		So(alertSettings, ShouldResemble, alertCfg)
	})
}
