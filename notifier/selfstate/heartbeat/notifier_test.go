package heartbeat

import (
	"errors"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewNotifierHeartbeater(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test NewNotifierHeartbeater", t, func() {
		Convey("With correct local checker heartbeater config", func() {
			cfg := NotifierHeartbeaterConfig{}

			expected := &notifierHeartbeater{
				heartbeaterBase: heartbeaterBase,
				cfg:             cfg,
			}

			notifierHeartbeater, err := NewNotifierHeartbeater(cfg, heartbeaterBase)
			So(err, ShouldBeNil)
			So(notifierHeartbeater, ShouldResemble, expected)
		})
	})
}

func TestNotifierHeartbeaterCheck(t *testing.T) {
	database, _, _, heartbeaterBase := heartbeaterHelper(t)

	cfg := NotifierHeartbeaterConfig{}

	notifierHeartbeater, _ := NewNotifierHeartbeater(cfg, heartbeaterBase)

	testErr := errors.New("test error")

	Convey("Test notifierHeartbeater.Check", t, func() {
		Convey("With GetNotifierState error", func() {
			database.EXPECT().GetNotifierState().Return(string(moira.SelfStateOK), testErr)

			state, err := notifierHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With notifier state equals error", func() {
			database.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)

			state, err := notifierHeartbeater.Check()
			So(err, ShouldResemble, nil)
			So(state, ShouldResemble, StateError)
		})

		Convey("With notifier state equals ok", func() {
			database.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)

			state, err := notifierHeartbeater.Check()
			So(err, ShouldResemble, nil)
			So(state, ShouldResemble, StateOK)
		})
	})
}

func TestNotifierHeartbeaterNeedTurnOffNotifier(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test notifierHeartbeater.TurnOffNotifier", t, func() {
		cfg := NotifierHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedTurnOffNotifier: true,
			},
		}

		notifierHeartbeater, err := NewNotifierHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needTurnOffNotifier := notifierHeartbeater.NeedTurnOffNotifier()
		So(needTurnOffNotifier, ShouldBeTrue)
	})
}

func TestNotifierHeartbeaterType(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test notifierHeartbeater.Type", t, func() {
		cfg := NotifierHeartbeaterConfig{}

		notifierHeartbeater, err := NewNotifierHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		notifierHeartbeaterType := notifierHeartbeater.Type()
		So(notifierHeartbeaterType, ShouldResemble, datatypes.HeartbeatNotifier)
	})
}

func TestNotifierHeartbeaterAlertSettings(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test notifierHeartbeater.AlertSettings", t, func() {
		alertCfg := AlertConfig{
			Name: "test name",
			Desc: "test desc",
		}

		cfg := NotifierHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				AlertCfg: alertCfg,
			},
		}

		notifierHeartbeater, err := NewNotifierHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		alertSettings := notifierHeartbeater.AlertSettings()
		So(alertSettings, ShouldResemble, alertCfg)
	})
}
