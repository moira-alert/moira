package selfstate

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_controller "github.com/moira-alert/moira/mock/controller"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_monitor "github.com/moira-alert/moira/mock/monitor"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/controller"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	"github.com/moira-alert/moira/notifier/selfstate/monitor"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSelfstateWorker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)
	dummyRegistry := metrics.NewDummyRegistry()
	heartbeatMetrics := metrics.ConfigureHeartBeatMetrics(dummyRegistry)

	cfg := Config{
		Enabled:       true,
		MonitorCfg:    MonitorConfig{},
		ControllerCfg: controller.ControllerConfig{},
	}

	Convey("Test NewSelfstateWorker", t, func() {
		worker, err := NewSelfstateWorker(cfg, logger, mockDatabase, mockNotifier, mockClock, heartbeatMetrics)
		So(err, ShouldBeNil)
		So(worker.monitors, ShouldHaveLength, 0)
	})
}

func TestCreateMonitors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test createMonitors", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		Convey("With disabled monitors", func() {
			monitors := createMonitors(MonitorConfig{}, logger, mockDatabase, mockClock, mockNotifier)
			So(monitors, ShouldHaveLength, 0)
		})

		Convey("With enabled user monitor", func() {
			cfg := MonitorConfig{
				UserCfg: monitor.UserMonitorConfig{
					MonitorBaseConfig: monitor.MonitorBaseConfig{
						Enabled: true,
						HeartbeatersCfg: heartbeat.HeartbeatersConfig{
							DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
								HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
									Enabled: true,
								},
								RedisDisconnectDelay: time.Minute,
							},
						},
						NoticeInterval: time.Minute,
						CheckInterval:  time.Minute,
					},
				},
			}

			monitors := createMonitors(cfg, logger, mockDatabase, mockClock, mockNotifier)
			So(monitors, ShouldHaveLength, 1)
		})

		Convey("With enabled user and admin monitors", func() {
			mockNotifier.EXPECT().GetSenders().Return(map[string]bool{})

			cfg := MonitorConfig{
				UserCfg: monitor.UserMonitorConfig{
					MonitorBaseConfig: monitor.MonitorBaseConfig{
						Enabled: true,
						HeartbeatersCfg: heartbeat.HeartbeatersConfig{
							DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
								HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
									Enabled: true,
								},
								RedisDisconnectDelay: time.Minute,
							},
						},
						NoticeInterval: time.Minute,
						CheckInterval:  time.Minute,
					},
				},
				AdminCfg: monitor.AdminMonitorConfig{
					MonitorBaseConfig: monitor.MonitorBaseConfig{
						Enabled: true,
						HeartbeatersCfg: heartbeat.HeartbeatersConfig{
							DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
								HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
									Enabled: true,
								},
								RedisDisconnectDelay: time.Minute,
							},
						},
						NoticeInterval: time.Minute,
						CheckInterval:  time.Minute,
					},
					AdminContacts: []map[string]string{},
				},
			}

			monitors := createMonitors(cfg, logger, mockDatabase, mockClock, mockNotifier)
			So(monitors, ShouldHaveLength, 2)
		})
	})
}

func TestCreateController(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)
	dummyRegistry := metrics.NewDummyRegistry()
	heartbeatMetrics := metrics.ConfigureHeartBeatMetrics(dummyRegistry)

	Convey("Test createController", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		Convey("With disabled controller", func() {
			controller := createController(controller.ControllerConfig{}, logger, mockDatabase, mockClock, heartbeatMetrics)
			So(controller, ShouldBeNil)
		})

		Convey("With enabled controller", func() {
			cfg := controller.ControllerConfig{
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

			controller := createController(cfg, logger, mockDatabase, mockClock, heartbeatMetrics)
			So(controller, ShouldNotBeNil)
		})
	})
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUserMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockAdminMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockController := mock_controller.NewMockController(mockCtrl)

	worker := &selfstateWorker{
		monitors:   []monitor.Monitor{mockUserMonitor, mockAdminMonitor},
		controller: mockController,
	}

	Convey("Test Start", t, func() {
		mockUserMonitor.EXPECT().Start()
		mockAdminMonitor.EXPECT().Start()
		mockController.EXPECT().Start()

		worker.Start()
	})
}

func TestStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUserMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockAdminMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockController := mock_controller.NewMockController(mockCtrl)

	userMonitorErr := errors.New("test user monitor error")
	adminMonitorErr := errors.New("test admin monitor error")
	controllerErr := errors.New("test controller error")

	worker := &selfstateWorker{
		monitors:   []monitor.Monitor{mockUserMonitor, mockAdminMonitor},
		controller: mockController,
	}

	Convey("Test Stop", t, func() {
		Convey("Without any errors", func() {
			mockUserMonitor.EXPECT().Stop().Return(nil)
			mockAdminMonitor.EXPECT().Stop().Return(nil)
			mockController.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldBeNil)
		})

		Convey("With user monitor error", func() {
			mockUserMonitor.EXPECT().Stop().Return(userMonitorErr)
			mockAdminMonitor.EXPECT().Stop().Return(nil)
			mockController.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(userMonitorErr))
		})

		Convey("With admin monitor error", func() {
			mockUserMonitor.EXPECT().Stop().Return(nil)
			mockAdminMonitor.EXPECT().Stop().Return(adminMonitorErr)
			mockController.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(adminMonitorErr))
		})

		Convey("With controller error", func() {
			mockUserMonitor.EXPECT().Stop().Return(nil)
			mockAdminMonitor.EXPECT().Stop().Return(nil)
			mockController.EXPECT().Stop().Return(controllerErr)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(controllerErr))
		})

		Convey("With admin and user monitor errors", func() {
			mockUserMonitor.EXPECT().Stop().Return(userMonitorErr)
			mockAdminMonitor.EXPECT().Stop().Return(adminMonitorErr)
			mockController.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(userMonitorErr, adminMonitorErr))
		})

		Convey("With admin, user monitors and controller errors", func() {
			mockUserMonitor.EXPECT().Stop().Return(userMonitorErr)
			mockAdminMonitor.EXPECT().Stop().Return(adminMonitorErr)
			mockController.EXPECT().Stop().Return(controllerErr)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(userMonitorErr, adminMonitorErr, controllerErr))
		})
	})
}
