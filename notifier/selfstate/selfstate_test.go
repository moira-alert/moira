package selfstate

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_monitor "github.com/moira-alert/moira/mock/monitor"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
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

	cfg := Config{
		Enabled:    true,
		MonitorCfg: MonitorConfig{},
	}

	Convey("Test NewSelfstateWorker", t, func() {
		worker, err := NewSelfstateWorker(cfg, logger, mockDatabase, mockNotifier, mockClock)
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

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUserMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockAdminMonitor := mock_monitor.NewMockMonitor(mockCtrl)

	worker := &selfstateWorker{
		monitors: []monitor.Monitor{mockUserMonitor, mockAdminMonitor},
	}

	Convey("Test Start", t, func() {
		mockUserMonitor.EXPECT().Start()
		mockAdminMonitor.EXPECT().Start()

		worker.Start()
	})
}

func TestStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUserMonitor := mock_monitor.NewMockMonitor(mockCtrl)
	mockAdminMonitor := mock_monitor.NewMockMonitor(mockCtrl)

	userMonitorErr := errors.New("test user monitor error")
	adminMonitorErr := errors.New("test admin monitor error")

	worker := &selfstateWorker{
		monitors: []monitor.Monitor{mockUserMonitor, mockAdminMonitor},
	}

	Convey("Test Stop", t, func() {
		Convey("Without any errors", func() {
			mockUserMonitor.EXPECT().Stop().Return(nil)
			mockAdminMonitor.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldBeNil)
		})

		Convey("With user monitor error", func() {
			mockUserMonitor.EXPECT().Stop().Return(userMonitorErr)
			mockAdminMonitor.EXPECT().Stop().Return(nil)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(userMonitorErr))
		})

		Convey("With admin monitor error", func() {
			mockUserMonitor.EXPECT().Stop().Return(nil)
			mockAdminMonitor.EXPECT().Stop().Return(adminMonitorErr)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(adminMonitorErr))
		})

		Convey("With admin and user monitor errors", func() {
			mockUserMonitor.EXPECT().Stop().Return(userMonitorErr)
			mockAdminMonitor.EXPECT().Stop().Return(adminMonitorErr)

			err := worker.Stop()
			So(err, ShouldResemble, errors.Join(userMonitorErr, adminMonitorErr))
		})
	})
}
