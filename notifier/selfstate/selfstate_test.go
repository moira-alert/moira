package selfstate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	mock_heartbeat "github.com/moira-alert/moira/mock/heartbeat"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

type selfCheckWorkerMock struct {
	selfCheckWorker *SelfCheckWorker
	database        *mock_moira_alert.MockDatabase
	notif           *mock_notifier.MockNotifier
	conf            Config
	mockCtrl        *gomock.Controller
}

func TestSelfCheckWorker_selfStateChecker(t *testing.T) {
	defaultLocalCluster := moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster)
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	mock := configureWorker(t, true)
	Convey("SelfCheckWorker should call all heartbeats checks", t, func() {
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil).Times(2)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil).Times(2)
		mock.database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), nil).Times(2)
		mock.database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(1), nil)

		// Start worker after configuring Mock to avoid race conditions
		err := mock.selfCheckWorker.Start()
		So(err, ShouldBeNil)

		So(len(mock.selfCheckWorker.heartbeatsGraph[0]), ShouldEqual, 1)
		So(len(mock.selfCheckWorker.heartbeatsGraph[1]), ShouldEqual, 4)

		const oneTickDelay = time.Millisecond * 1500

		time.Sleep(oneTickDelay) // wait for one tick of worker

		err = mock.selfCheckWorker.Stop()
		So(err, ShouldBeNil)
	})

	mock.mockCtrl.Finish()
}

func TestSelfCheckWorker_sendMessages(t *testing.T) {
	Convey("Should call notifier send", t, func() {
		mock := configureWorker(t, true)
		err := mock.selfCheckWorker.Start()
		So(err, ShouldBeNil)

		mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

		var events []heartbeatNotificationEvent

		mock.selfCheckWorker.sendMessages(events)

		err = mock.selfCheckWorker.Stop()
		So(err, ShouldBeNil)
		mock.mockCtrl.Finish()
	})
}

func TestSelfCheckWorker_constructUserNotification(t *testing.T) {
	Convey("Should resemble events to contacts trought system tags", t, func() {
		contact := moira.ContactData{
			ID:    "some-contact",
			Type:  "my_type",
			Value: "123",
		}

		notifAndTags := []heartbeatNotificationEvent{
			{
				NotificationEvent: moira.NotificationEvent{
					Metric: "Triggered!!!",
				},
				CheckTags: heartbeat.CheckTags{
					"sys-tag1",
				},
			},
			{
				NotificationEvent: moira.NotificationEvent{
					Metric: "Some another problem!!!",
				},
				CheckTags: heartbeat.CheckTags{
					"sys-tag2", "sys-tag-common",
				},
			},
		}

		expected := []*notifier.NotificationPackage{
			{
				Contact: contact,
				Trigger: moira.TriggerData{
					Name:       "Moira health check",
					ErrorValue: float64(0),
				},
				Events: []moira.NotificationEvent{
					{
						Metric: "Triggered!!!",
					},
					{
						Metric: "Some another problem!!!",
					},
				},
				DontResend: true,
			},
		}

		mockCtrl := gomock.NewController(t)
		database := mock_moira_alert.NewMockDatabase(mockCtrl)

		database.EXPECT().GetTagsSubscriptions([]string{"sys-tag1"}).Return([]*moira.SubscriptionData{
			{
				ID:       "sub-1",
				Contacts: []string{contact.ID},
			},
		}, nil)
		database.EXPECT().GetTagsSubscriptions([]string{"sys-tag2", "sys-tag-common"}).Return([]*moira.SubscriptionData{
			{
				ID:       "sub-2",
				Contacts: []string{contact.ID},
			},
		}, nil)

		database.EXPECT().GetContacts([]string{contact.ID}).Return([]*moira.ContactData{
			&contact,
		}, nil).Times(2)

		logger, _ := logging.GetLogger("SelfState")
		notif := mock_notifier.NewMockNotifier(mockCtrl)

		mock := &selfCheckWorkerMock{
			selfCheckWorker: NewSelfCheckWorker(logger, database, notif, Config{}),
			mockCtrl:        mockCtrl,
		}

		actual, err := mock.selfCheckWorker.constructUserNotification(notifAndTags)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
		mock.mockCtrl.Finish()
	})
}

func TestSelfCheckWorker_enableNotifierIfNeed(t *testing.T) {
	mock := configureWorker(t, false)

	Convey("Should enable if notifier is disabled by auto", t, func() {
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateOK)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldBeNil)
		So(notifierEnabled, ShouldBeTrue)
	})

	Convey("Should switch notifier to AUTO if enabled manually", t, func() {
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}, nil)

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateOK)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldBeNil)
		So(notifierEnabled, ShouldBeTrue)
	})

	Convey("Should not enable if notifier is disabled manually", t, func() {
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorManual,
		}, nil)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldBeNil)
		So(notifierEnabled, ShouldBeFalse)
	})

	Convey("Should not enable notifier if it is already enabled", t, func() {
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldBeNil)
		So(notifierEnabled, ShouldBeFalse)
	})

	Convey("Should not enable notifier if getting state throw error", t, func() {
		expected_err := fmt.Errorf("error")
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{}, expected_err)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldResemble, expected_err)
		So(notifierEnabled, ShouldBeFalse)
	})

	Convey("Should not enable notifier if notifier enabling returns error", t, func() {
		expected_err := fmt.Errorf("error")

		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateOK).Return(expected_err)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		So(err, ShouldEqual, expected_err)
		So(notifierEnabled, ShouldBeFalse)
	})
}

func TestSelfCheckWorker_Start(t *testing.T) {
	mock := configureWorker(t, false)
	Convey("When Contact not corresponds to any Sender", t, func() {
		mock.notif.EXPECT().GetSenders().Return(nil)

		Convey("Start should return error", func() {
			err := mock.selfCheckWorker.Start()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSelfCheckWorker(t *testing.T) {
	Convey("Test checked heartbeat", t, func() {
		err := errors.New("test error")
		now := time.Now().Unix()

		mock := configureWorker(t, false)

		Convey("Test handle error and no needed send events", func() {
			check := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{{check}}

			mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
				Actor: moira.SelfStateActorAutomatic,
				State: moira.SelfStateOK,
			}, nil)

			check.EXPECT().Check(now).Return(int64(0), false, err)

			events := mock.selfCheckWorker.handleCheckServices(now)
			So(events, ShouldBeNil)
		})

		Convey("Test turn off notification", func() {
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{
				{first},
				{second},
			}

			first.EXPECT().NeedTurnOffNotifier().Return(true)
			first.EXPECT().GetErrorMessage().Return(moira.SelfStateERROR)
			first.EXPECT().Check(now).Return(int64(0), true, nil)
			first.EXPECT().GetCheckTags().Return([]string{})
			mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR)

			events := mock.selfCheckWorker.handleCheckServices(now)
			So(len(events), ShouldEqual, 1)
		})

		Convey("Test turn on notification", func() {
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{
				{first},
				{second},
			}

			first.EXPECT().Check(now).Return(int64(15), false, nil)
			second.EXPECT().Check(now).Return(int64(15), false, nil)

			mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
				State: moira.SelfStateERROR,
				Actor: moira.SelfStateActorAutomatic,
			}, nil)
			mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateOK)

			events := mock.selfCheckWorker.handleCheckServices(now)
			So(len(events), ShouldEqual, 0)
		})

		Convey("Test of sending notifications from a check", func() {
			now = time.Now().Unix()
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{
				{first},
				{second},
			}
			nextSendErrorMessage := time.Now().Unix() - time.Hour.Milliseconds()

			first.EXPECT().Check(now).Return(int64(0), true, nil)
			first.EXPECT().GetErrorMessage().Return(moira.SelfStateERROR)
			first.EXPECT().NeedTurnOffNotifier().Return(true)
			first.EXPECT().GetCheckTags().Return([]string{})
			mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR).Return(err)
			mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

			nextSendErrorMessage = mock.selfCheckWorker.check(now, nextSendErrorMessage)
			So(nextSendErrorMessage, ShouldEqual, now+60)
		})

		mock.mockCtrl.Finish()
	})
}

func configureWorker(t *testing.T, isStart bool) *selfCheckWorkerMock {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}
	conf := Config{
		Enabled: true,
		Contacts: []map[string]string{
			adminContact,
		},
		RedisDisconnectDelaySeconds:    10,
		LastMetricReceivedDelaySeconds: 60,
		LastCheckDelaySeconds:          120,
		NoticeIntervalSeconds:          60,
		LastRemoteCheckDelaySeconds:    120,
		CheckInterval:                  1 * time.Second,
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)

	if isStart {
		senders := map[string]bool{
			"admin-mail": true,
		}
		notif.EXPECT().GetSenders().Return(senders).MinTimes(1)

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil)
		lock.EXPECT().Release()
		database.EXPECT().NewLock(gomock.Any(), gomock.Any()).Return(lock)
	}

	return &selfCheckWorkerMock{
		selfCheckWorker: NewSelfCheckWorker(logger, database, notif, conf),
		database:        database,
		notif:           notif,
		conf:            conf,
		mockCtrl:        mockCtrl,
	}
}
