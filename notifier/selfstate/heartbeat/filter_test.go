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
	defaultMetricReceivedDelay = time.Minute
)

func TestNewFilterHeartbeater(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	validationErr := validator.ValidationErrors{}

	Convey("Test NewFilterHeartbeater", t, func() {
		Convey("With too low metric received delay", func() {
			cfg := FilterHeartbeaterConfig{
				MetricReceivedDelay: -1,
			}

			filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(filterHeartbeater, ShouldBeNil)
		})

		Convey("Without metric received delay", func() {
			cfg := FilterHeartbeaterConfig{
				HeartbeaterBaseConfig: HeartbeaterBaseConfig{
					Enabled: true,
				},
			}

			filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
			So(errors.As(err, &validationErr), ShouldBeTrue)
			So(filterHeartbeater, ShouldBeNil)
		})

		Convey("With correct filter heartbeater config", func() {
			cfg := FilterHeartbeaterConfig{
				MetricReceivedDelay: 1,
			}

			expected := &filterHeartbeater{
				heartbeaterBase: heartbeaterBase,
				cfg:             cfg,
			}

			filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
			So(err, ShouldBeNil)
			So(filterHeartbeater, ShouldResemble, expected)
		})
	})
}

func TestFilterHeartbeaterCheck(t *testing.T) {
	database, clock, testTime, heartbeaterBase := heartbeaterHelper(t)

	cfg := FilterHeartbeaterConfig{
		MetricReceivedDelay: defaultMetricReceivedDelay,
	}

	filterHeartbeater, _ := NewFilterHeartbeater(cfg, heartbeaterBase)

	var (
		testErr                                         = errors.New("test error")
		triggersToCheckCount, metricsUpdatesCount int64 = 10, 10
	)

	Convey("Test filterHeartbeater.Check", t, func() {
		Convey("With GetTriggersToCheckCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, testErr)

			state, err := filterHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With GetMetricsUpdatesCount error", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetMetricsUpdatesCount().Return(metricsUpdatesCount, testErr)

			state, err := filterHeartbeater.Check()
			So(err, ShouldResemble, testErr)
			So(state, ShouldResemble, StateError)
		})

		Convey("With last metrics count not equal current metrics count", func() {
			defer func() {
				filterHeartbeater.lastMetricsCount = 0
			}()

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetMetricsUpdatesCount().Return(metricsUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := filterHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(filterHeartbeater.lastMetricsCount, ShouldResemble, metricsUpdatesCount)
		})

		Convey("With zero triggers to check count", func() {
			defer func() {
				filterHeartbeater.lastMetricsCount = 0
			}()

			var zeroTriggersToCheckCount int64

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(zeroTriggersToCheckCount, nil)
			database.EXPECT().GetMetricsUpdatesCount().Return(metricsUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := filterHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
			So(filterHeartbeater.lastMetricsCount, ShouldResemble, metricsUpdatesCount)
		})

		filterHeartbeater.lastMetricsCount = metricsUpdatesCount

		Convey("With too much time elapsed since the last successful check", func() {
			filterHeartbeater.lastSuccessfulCheck = testTime.Add(-10 * defaultMetricReceivedDelay)
			defer func() {
				filterHeartbeater.lastSuccessfulCheck = testTime
			}()

			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetMetricsUpdatesCount().Return(metricsUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := filterHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateError)
		})

		Convey("With short time elapsed since the last successful check", func() {
			database.EXPECT().GetTriggersToCheckCount(localClusterKey).Return(triggersToCheckCount, nil)
			database.EXPECT().GetMetricsUpdatesCount().Return(metricsUpdatesCount, nil)
			clock.EXPECT().NowUTC().Return(testTime)

			state, err := filterHeartbeater.Check()
			So(err, ShouldBeNil)
			So(state, ShouldResemble, StateOK)
		})
	})
}

func TestFilterHeartbeaterNeedTurnOffNotifier(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test filterHeartbeater.TurnOffNotifier", t, func() {
		cfg := FilterHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				NeedTurnOffNotifier: true,
			},
			MetricReceivedDelay: defaultMetricReceivedDelay,
		}

		filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		needTurnOffNotifier := filterHeartbeater.NeedTurnOffNotifier()
		So(needTurnOffNotifier, ShouldBeTrue)
	})
}

func TestFilterHeartbeaterType(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test filterHeartbeater.Type", t, func() {
		cfg := FilterHeartbeaterConfig{
			MetricReceivedDelay: defaultMetricReceivedDelay,
		}

		filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		filterHeartbeaterType := filterHeartbeater.Type()
		So(filterHeartbeaterType, ShouldResemble, datatypes.HeartbeatFilter)
	})
}

func TestFilterHeartbeaterAlertSettings(t *testing.T) {
	_, _, _, heartbeaterBase := heartbeaterHelper(t)

	Convey("Test filterHeartbeater.AlertSettings", t, func() {
		alertCfg := AlertConfig{
			Name: "test name",
			Desc: "test desc",
		}

		cfg := FilterHeartbeaterConfig{
			HeartbeaterBaseConfig: HeartbeaterBaseConfig{
				AlertCfg: alertCfg,
			},
			MetricReceivedDelay: defaultMetricReceivedDelay,
		}

		filterHeartbeater, err := NewFilterHeartbeater(cfg, heartbeaterBase)
		So(err, ShouldBeNil)

		alertSettings := filterHeartbeater.AlertSettings()
		So(alertSettings, ShouldResemble, alertCfg)
	})
}
