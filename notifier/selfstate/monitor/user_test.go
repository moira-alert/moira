package monitor

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestNewForUser(t *testing.T) {
	t.Skip()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockLogger := mock_moira_alert.NewMockLogger(mockCtrl)
	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockClock := mock_clock.NewMockClock(mockCtrl)
	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test NewForUser", t, func() {
		userCfg := UserMonitorConfig{
			MonitorBaseConfig: MonitorBaseConfig{
				Enabled:         true,
				HeartbeatersCfg: defaultHeartbeatersConfig,
				NoticeInterval:  time.Minute,
				CheckInterval:   time.Minute,
			},
		}
		userMonitorCfg := monitorConfig{
			Name:           userMonitorName,
			LockName:       userMonitorLockName,
			LockTTL:        userMonitorLockTTL,
			NoticeInterval: time.Minute,
			CheckInterval:  time.Minute,
		}
		um := userMonitor{
			userCfg:  userCfg,
			database: mockDatabase,
			notifier: mockNotifier,
		}
		mockClock.EXPECT().NowUTC().Return(testTime).Times(2)
		heartbeaters := createHearbeaters(userCfg.HeartbeatersCfg, mockLogger, mockDatabase, mockClock)
		hearbeatersInfo := make(map[datatypes.HeartbeatType]*hearbeatInfo, len(heartbeaters))
		for _, heartbeater := range heartbeaters {
			hearbeatersInfo[heartbeater.Type()] = &hearbeatInfo{
				lastCheckState: heartbeat.StateOK,
			}
		}

		userMonitor, err := NewForUser(userCfg, mockLogger, mockDatabase, mockClock, mockNotifier)
		So(err, ShouldNotBeNil)
		So(userMonitor, ShouldResemble, &monitor{
			cfg:               userMonitorCfg,
			logger:            mockLogger,
			database:          mockDatabase,
			notifier:          mockNotifier,
			heartbeaters:      heartbeaters,
			clock:             mockClock,
			heartbeatsInfo:    hearbeatersInfo,
			sendNotifications: um.sendNotifications,
		})
	})
}

func TestUserSendNotifications(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)

	sendingWG := &sync.WaitGroup{}

	userMonitor := userMonitor{
		notifier: mockNotifier,
		database: mockDatabase,
	}

	contactIDs := []string{"test-contact-id"}
	contact := &moira.ContactData{
		ID: "test-contact-id",
	}
	contacts := []*moira.ContactData{contact}

	testErr := errors.New("test error")

	Convey("Test sendNotifications", t, func() {
		Convey("With empty notification packages", func() {
			pkgs := []notifier.NotificationPackage{}
			err := userMonitor.sendNotifications(pkgs)
			So(err, ShouldBeNil)
		})

		Convey("With GetHeartbeatTypeContactIDs error", func() {
			pkgs := []notifier.NotificationPackage{
				{
					Events: []moira.NotificationEvent{
						{
							Metric: string(datatypes.HeartbeatNotifier),
						},
					},
				},
			}
			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatNotifier).Return([]string{}, testErr).Times(1)
			err := userMonitor.sendNotifications(pkgs)
			So(err, ShouldResemble, fmt.Errorf("failed to get heartbeat type contact ids: %w", testErr))
		})

		Convey("With GetContacts error", func() {
			pkgs := []notifier.NotificationPackage{
				{
					Events: []moira.NotificationEvent{
						{
							Metric: string(datatypes.HeartbeatNotifier),
						},
					},
				},
			}
			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatNotifier).Return(contactIDs, nil).Times(1)
			mockDatabase.EXPECT().GetContacts(contactIDs).Return(nil, testErr).Times(1)
			err := userMonitor.sendNotifications(pkgs)
			So(err, ShouldResemble, fmt.Errorf("failed to get contacts by ids: %w", testErr))
		})

		Convey("With correct sending notification packages", func() {
			pkgs := []notifier.NotificationPackage{
				{
					Events: []moira.NotificationEvent{
						{
							Metric: string(datatypes.HeartbeatNotifier),
						},
					},
				},
			}
			pkgWithContact := &notifier.NotificationPackage{
				Events: []moira.NotificationEvent{
					{
						Metric: string(datatypes.HeartbeatNotifier),
					},
				},
				Contact: *contacts[0],
			}
			mockDatabase.EXPECT().GetHeartbeatTypeContactIDs(datatypes.HeartbeatNotifier).Return(contactIDs, nil).Times(1)
			mockDatabase.EXPECT().GetContacts(contactIDs).Return(contacts, nil).Times(1)
			mockNotifier.EXPECT().Send(pkgWithContact, sendingWG).Times(1)
			err := userMonitor.sendNotifications(pkgs)
			So(err, ShouldBeNil)
		})
	})
}
