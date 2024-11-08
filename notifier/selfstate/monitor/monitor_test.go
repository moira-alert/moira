package monitor

import (
	"sync"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_heartbeat "github.com/moira-alert/moira/mock/heartbeat"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

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
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldBeEmpty)
		})

		Convey("With database heartbeater", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 1)
		})

		Convey("With database and filter heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					MetricReceivedDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 2)
		})

		Convey("With database, filter and local checker heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					LocalCheckDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 3)
		})

		Convey("With database, filter, local checker and remote checker heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					LocalCheckDelay: time.Minute,
				},
				RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RemoteCheckDelay: time.Minute,
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 4)
		})

		Convey("With database, filter, local checker, remote checker and notifier heartbeaters", func() {
			hbCfg := heartbeat.HeartbeatersConfig{
				DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RedisDisconnectDelay: time.Minute,
				},
				FilterCfg: heartbeat.FilterHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					MetricReceivedDelay: time.Minute,
				},
				LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					LocalCheckDelay: time.Minute,
				},
				RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
					RemoteCheckDelay: time.Minute,
				},
				NotifierCfg: heartbeat.NotifierHeartbeaterConfig{
					HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
						Enabled: true,
					},
				},
			}
			mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()
			heartbeaters := createHearbeaters(hbCfg, mockLogger, mockDatabase, mockClock)
			So(heartbeaters, ShouldHaveLength, 5)
		})
	})
}

func TestCreateErrorNotificationPackage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockLogger := mock_moira_alert.NewMockLogger(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test createErrorNotificationPackage", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).Times(2)

		m := monitor{
			heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase: {},
			},
		}

		heartbeaterBase := heartbeat.NewHeartbeaterBase(mockLogger, mockDatabase, mockClock)
		databaseHeartbeater, err := heartbeat.NewDatabaseHeartbeater(heartbeat.DatabaseHeartbeaterConfig{
			HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
				AlertCfg: heartbeat.AlertConfig{
					Name: "Database Heartbeater",
					Desc: "Some Database problems",
				},
			},
		}, heartbeaterBase)
		So(err, ShouldBeNil)

		expectedPkg := &notifier.NotificationPackage{
			Events: []moira.NotificationEvent{
				{
					Timestamp: testTime.Unix(),
					OldState:  moira.StateNODATA,
					State:     moira.StateERROR,
					Metric:    string(databaseHeartbeater.Type()),
					Value:     &errorValue,
				},
			},
			Trigger: moira.TriggerData{
				Name:       "Database Heartbeater",
				Desc:       "Some Database problems",
				ErrorValue: triggerErrorValue,
			},
		}

		pkg := m.createErrorNotificationPackage(databaseHeartbeater, mockClock)
		So(pkg, ShouldResemble, expectedPkg)
		So(m.heartbeatsInfo[databaseHeartbeater.Type()].lastAlertTime, ShouldEqual, testTime)
	})
}

func TestCheck(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)
	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)

	Convey("Test check", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		databaseHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		localCheckerHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		notifierHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)

		databaseHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil).AnyTimes()
		databaseHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatDatabase).AnyTimes()
		databaseHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Database Heartbeater",
			Desc: "Database Problems",
		}).AnyTimes()

		localCheckerHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil).AnyTimes()
		localCheckerHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatLocalChecker).AnyTimes()
		localCheckerHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Local Checker Heartbeater",
			Desc: "Local Checker Problems",
		}).AnyTimes()

		notifierHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil).AnyTimes()
		notifierHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatNotifier).AnyTimes()
		notifierHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Notifier Heartbeater",
			Desc: "Notifier Problems",
		}).AnyTimes()

		sendingWG := &sync.WaitGroup{}

		Convey("With admin monitor", func() {
			am := adminMonitor{
				adminCfg: AdminMonitorConfig{
					AdminContacts: []map[string]string{
						{
							"type":  "telegram",
							"value": "@webcamsmodel",
						},
					},
				},
				notifier: mockNotifier,
			}

			m := monitor{
				cfg: monitorConfig{
					NoticeInterval: time.Minute,
				},
				heartbeaters: []heartbeat.Heartbeater{databaseHeartbeater, localCheckerHeartbeater, notifierHeartbeater},
				heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
					datatypes.HeartbeatDatabase:     {},
					datatypes.HeartbeatLocalChecker: {},
					datatypes.HeartbeatNotifier:     {},
				},
				notifier:          mockNotifier,
				logger:            logger,
				clock:             mockClock,
				sendNotifications: am.sendNotifications,
			}

			contact := moira.ContactData{
				Type:  "telegram",
				Value: "@webcamsmodel",
			}

			databaseErrorPkg := m.createErrorNotificationPackage(databaseHeartbeater, mockClock)
			databaseErrorPkg.Contact = contact
			localCheckerErrorPkg := m.createErrorNotificationPackage(localCheckerHeartbeater, mockClock)
			localCheckerErrorPkg.Contact = contact
			notifierErrorPkg := m.createErrorNotificationPackage(notifierHeartbeater, mockClock)
			notifierErrorPkg.Contact = contact

			m.heartbeatsInfo = map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
				datatypes.HeartbeatLocalChecker: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
				datatypes.HeartbeatNotifier: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
			}

			mockNotifier.EXPECT().Send(databaseErrorPkg, sendingWG)
			mockNotifier.EXPECT().Send(localCheckerErrorPkg, sendingWG)
			mockNotifier.EXPECT().Send(notifierErrorPkg, sendingWG)

			m.check()
		})

		Convey("With user monitor", func() {
			um := userMonitor{
				notifier: mockNotifier,
				database: mockDatabase,
			}

			m := monitor{
				cfg: monitorConfig{
					NoticeInterval: time.Minute,
				},
				heartbeaters: []heartbeat.Heartbeater{databaseHeartbeater, localCheckerHeartbeater, notifierHeartbeater},
				heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
					datatypes.HeartbeatDatabase:     {},
					datatypes.HeartbeatLocalChecker: {},
					datatypes.HeartbeatNotifier:     {},
				},
				notifier:          mockNotifier,
				logger:            logger,
				clock:             mockClock,
				sendNotifications: um.sendNotifications,
			}

			contactIDs := []string{"test-contact-id"}
			contact := moira.ContactData{
				Type:  "telegram",
				Value: "@webcamsmodel",
			}
			contacts := []*moira.ContactData{&contact}

			databaseErrorPkg := m.createErrorNotificationPackage(databaseHeartbeater, mockClock)
			databaseErrorPkg.Contact = contact
			localCheckerErrorPkg := m.createErrorNotificationPackage(localCheckerHeartbeater, mockClock)
			localCheckerErrorPkg.Contact = contact
			notifierErrorPkg := m.createErrorNotificationPackage(notifierHeartbeater, mockClock)
			notifierErrorPkg.Contact = contact

			m.heartbeatsInfo = map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
				datatypes.HeartbeatLocalChecker: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
				datatypes.HeartbeatNotifier: {
					lastAlertTime: testTime.Add(-2 * time.Minute),
				},
			}

			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatDatabase).Return(contactIDs, nil)
			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatLocalChecker).Return(contactIDs, nil)
			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatNotifier).Return(contactIDs, nil)

			mockDatabase.EXPECT().GetContacts(contactIDs).Return(contacts, nil).AnyTimes()

			mockNotifier.EXPECT().Send(databaseErrorPkg, sendingWG)
			mockNotifier.EXPECT().Send(localCheckerErrorPkg, sendingWG)
			mockNotifier.EXPECT().Send(notifierErrorPkg, sendingWG)

			m.check()
		})
	})
}

func TestCheckHeartbeats(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := mock_clock.NewMockClock(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test checkHeartbeats", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		databaseHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		localCheckerHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)
		notifierHeartbeater := mock_heartbeat.NewMockHeartbeater(mockCtrl)

		m := monitor{
			cfg: monitorConfig{
				NoticeInterval: time.Minute,
			},
			heartbeaters: []heartbeat.Heartbeater{databaseHeartbeater, localCheckerHeartbeater, notifierHeartbeater},
			heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase:     {},
				datatypes.HeartbeatLocalChecker: {},
				datatypes.HeartbeatNotifier:     {},
			},
			logger: logger,
			clock:  mockClock,
		}

		databaseHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil)
		databaseHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatDatabase).AnyTimes()
		databaseHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Database Heartbeater",
			Desc: "Database Problems",
		}).AnyTimes()

		localCheckerHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil)
		localCheckerHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatLocalChecker).AnyTimes()
		localCheckerHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Local Checker Heartbeater",
			Desc: "Local Checker Problems",
		}).AnyTimes()

		notifierHeartbeater.EXPECT().Check().Return(heartbeat.StateError, nil)
		notifierHeartbeater.EXPECT().Type().Return(datatypes.HeartbeatNotifier).AnyTimes()
		notifierHeartbeater.EXPECT().AlertSettings().Return(heartbeat.AlertConfig{
			Name: "Notifier Heartbeater",
			Desc: "Notifier Problems",
		}).AnyTimes()

		databaseErrorPkg := m.createErrorNotificationPackage(databaseHeartbeater, mockClock)
		localCheckerErrorPkg := m.createErrorNotificationPackage(localCheckerHeartbeater, mockClock)
		notifierErrorPkg := m.createErrorNotificationPackage(notifierHeartbeater, mockClock)
		expectedPkgs := []notifier.NotificationPackage{*databaseErrorPkg, *localCheckerErrorPkg, *notifierErrorPkg}

		m.heartbeatsInfo = map[datatypes.HeartbeatType]*hearbeatInfo{
			datatypes.HeartbeatDatabase: {
				lastAlertTime: testTime.Add(-2 * time.Minute),
			},
			datatypes.HeartbeatLocalChecker: {
				lastAlertTime: testTime.Add(-2 * time.Minute),
			},
			datatypes.HeartbeatNotifier: {
				lastAlertTime: testTime.Add(-2 * time.Minute),
			},
		}

		pkgs := m.checkHeartbeats()
		So(pkgs, ShouldResemble, expectedPkgs)
		So(m.heartbeatsInfo, ShouldResemble, map[datatypes.HeartbeatType]*hearbeatInfo{
			datatypes.HeartbeatDatabase: {
				lastAlertTime:  testTime,
				lastCheckState: heartbeat.StateError,
			},
			datatypes.HeartbeatLocalChecker: {
				lastAlertTime:  testTime,
				lastCheckState: heartbeat.StateError,
			},
			datatypes.HeartbeatNotifier: {
				lastAlertTime:  testTime,
				lastCheckState: heartbeat.StateError,
			},
		})
	})
}

func TestGenerateHeartbeatNotificationPackage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockLogger := mock_moira_alert.NewMockLogger(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test generateHeartbeatNotificationPackage", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).AnyTimes()

		m := monitor{
			cfg: monitorConfig{
				NoticeInterval: time.Minute,
			},
			heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase: {},
			},
			clock: mockClock,
		}

		heartbeaterBase := heartbeat.NewHeartbeaterBase(mockLogger, mockDatabase, mockClock)
		databaseHeartbeater, err := heartbeat.NewDatabaseHeartbeater(heartbeat.DatabaseHeartbeaterConfig{}, heartbeaterBase)
		So(err, ShouldBeNil)

		Convey("isDegraded state and allowNotify", func() {
			m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateOK,
				lastAlertTime:  testTime.Add(-m.cfg.NoticeInterval - time.Minute),
			}
			defer func() {
				m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{}
			}()

			pkg := m.generateHeartbeatNotificationPackage(databaseHeartbeater, heartbeat.StateError)
			So(pkg, ShouldNotBeNil)
			So(pkg.Events[0].State, ShouldEqual, moira.StateERROR)
		})

		Convey("isDegraded state and not allowNotify", func() {
			m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateOK,
				lastAlertTime:  testTime.Add(-m.cfg.NoticeInterval + time.Minute),
			}
			defer func() {
				m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{}
			}()

			pkg := m.generateHeartbeatNotificationPackage(databaseHeartbeater, heartbeat.StateError)
			So(pkg, ShouldBeNil)
		})

		Convey("isRecovered state and allowNotify", func() {
			m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateError,
				lastAlertTime:  testTime.Add(-m.cfg.NoticeInterval - time.Minute),
			}
			defer func() {
				m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{}
			}()

			pkg := m.generateHeartbeatNotificationPackage(databaseHeartbeater, heartbeat.StateOK)
			So(pkg, ShouldNotBeNil)
			So(pkg.Events[0].State, ShouldEqual, moira.StateOK)
		})

		Convey("isRecovered state and not allowNotify", func() {
			m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateError,
				lastAlertTime:  testTime.Add(-m.cfg.NoticeInterval + time.Minute),
			}
			defer func() {
				m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{}
			}()

			pkg := m.generateHeartbeatNotificationPackage(databaseHeartbeater, heartbeat.StateOK)
			So(pkg, ShouldNotBeNil)
			So(pkg.Events[0].State, ShouldEqual, moira.StateOK)
		})

		Convey("not degraded and not recovered", func() {
			m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateOK,
				lastAlertTime:  testTime.Add(-m.cfg.NoticeInterval - time.Minute),
			}
			defer func() {
				m.heartbeatsInfo[databaseHeartbeater.Type()] = &hearbeatInfo{}
			}()

			pkg := m.generateHeartbeatNotificationPackage(databaseHeartbeater, heartbeat.StateOK)
			So(pkg, ShouldBeNil)
		})
	})
}

func TestCreateOkNotificationPackage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	mockLogger := mock_moira_alert.NewMockLogger(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test createOkNotificationPackage", t, func() {
		mockClock.EXPECT().NowUTC().Return(testTime).Times(2)

		m := monitor{
			heartbeatsInfo: map[datatypes.HeartbeatType]*hearbeatInfo{
				datatypes.HeartbeatDatabase: {},
			},
		}

		heartbeaterBase := heartbeat.NewHeartbeaterBase(mockLogger, mockDatabase, mockClock)
		databaseHeartbeater, err := heartbeat.NewDatabaseHeartbeater(heartbeat.DatabaseHeartbeaterConfig{
			HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
				AlertCfg: heartbeat.AlertConfig{
					Name: "Database Heartbeater",
					Desc: "Some Database problems",
				},
			},
		}, heartbeaterBase)
		So(err, ShouldBeNil)

		expectedPkg := &notifier.NotificationPackage{
			Events: []moira.NotificationEvent{
				{
					Timestamp: testTime.Unix(),
					OldState:  moira.StateERROR,
					State:     moira.StateOK,
					Metric:    string(databaseHeartbeater.Type()),
					Value:     &okValue,
				},
			},
			Trigger: moira.TriggerData{
				Name:       "Database Heartbeater",
				Desc:       "Some Database problems",
				ErrorValue: triggerErrorValue,
			},
		}

		pkg := m.createOkNotificationPackage(databaseHeartbeater, mockClock)
		So(pkg, ShouldResemble, expectedPkg)
		So(m.heartbeatsInfo[databaseHeartbeater.Type()].lastAlertTime, ShouldEqual, testTime)
	})
}
