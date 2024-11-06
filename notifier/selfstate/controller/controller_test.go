package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_heartbeat "github.com/moira-alert/moira/mock/heartbeat"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestNewController(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test NewController", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		Convey("With disabled config", func() {
			cfg := ControllerConfig{}

			c, err := NewController(cfg, logger, mockDatabase, mockClock)
			So(err, ShouldBeNil)
			So(c, ShouldResemble, &controller{
				cfg:          cfg,
				logger:       logger,
				database:     mockDatabase,
				heartbeaters: make([]heartbeat.Heartbeater, 0),
			})
		})

		Convey("With just enabled config", func() {
			cfg := ControllerConfig{
				Enabled: true,
			}

			c, err := NewController(cfg, logger, mockDatabase, mockClock)
			So(err, ShouldNotBeNil)
			So(c, ShouldBeNil)
		})

		Convey("With just enabled config and filled heartbeaters", func() {
			cfg := ControllerConfig{
				Enabled: true,
				HeartbeatersCfg: heartbeat.HeartbeatersConfig{
					DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
						HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
							Enabled: true,
						},
						RedisDisconnectDelay: time.Minute,
					},
				},
			}

			c, err := NewController(cfg, logger, mockDatabase, mockClock)
			So(err, ShouldNotBeNil)
			So(c, ShouldBeNil)
		})

		Convey("With enabled config, filled heartbeaters and set check interval", func() {
			cfg := ControllerConfig{
				Enabled: true,
				HeartbeatersCfg: heartbeat.HeartbeatersConfig{
					DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
						HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
							Enabled: true,
						},
						RedisDisconnectDelay: time.Minute,
					},
				},
				CheckInterval: time.Minute,
			}

			_, err := NewController(cfg, logger, mockDatabase, mockClock)
			So(err, ShouldBeNil)
		})
	})
}

func TestCreateHeartbeaters(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockLogger := mock_moira_alert.NewMockLogger(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test createHeartbeaters", t, func() {
		Convey("Without any heartbeater", func() {
			hbCfg := heartbeat.HeartbeatersConfig{}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldBeEmpty)
		})

		Convey("With just enabled database heartbeater", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldBeEmpty)
		})

		Convey("With database heartbeater", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 1)
		})

		Convey("With database and filter heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					MetricReceivedDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 2)
		})

		Convey("With database, filter and local checker heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					LocalCheckDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 3)
		})

		Convey("With database, filter, local checker and remote checker heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					LocalCheckDelay: time.Minute,
				},
				RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RemoteCheckDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 4)
		})

		Convey("With database, filter, local checker, remote checker and notifier heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					LocalCheckDelay: time.Minute,
				},
				RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
					RemoteCheckDelay: time.Minute,
				},
				NotifierCfg: heartbeat.NotifierHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled:             true,
						NeedTurnOffNotifier: true,
					},
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime)
			heartbeaters := createHeartbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 5)
		})
	})
}

func TestCheckHeartbeats(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test checkHeartbeats", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		databaseHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		localCheckerHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		notifierHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)

		c := &controller{
			heartbeaters: []heartbeat.Heartbeater{databaseHeartbeater, localCheckerHeartbeater, notifierHeartbeater},
			logger:       logger,
			database:     mockDatabase,
		}

		Convey("Without error heartbeat states", func() {
			databaseHeartbeater.EXPECT().Check().Return(heartbeat.StateOK, nil)
			localCheckerHeartbeater.EXPECT().Check().Return(heartbeat.StateOK, nil)
			notifierHeartbeater.EXPECT().Check().Return(heartbeat.StateOK, nil)

			c.checkHeartbeats()
		})

		Convey("With heartbeat error state", func() {
			databaseHeartbeater.EXPECT().Check().Return(heartbeat.StateOK, nil)
			localCheckerHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil)
			mockDatabase.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)

			c.checkHeartbeats()
		})

		Convey("With heartbeat error state and error while set notifier state", func() {
			dbErr := errors.New("test database error")
			databaseHeartbeater.EXPECT().Check().Return(heartbeat.StateOK, nil)
			localCheckerHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil)
			mockDatabase.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(dbErr)
			localCheckerHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatLocalChecker)

			c.checkHeartbeats()
		})
	})
}
