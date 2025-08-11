package selfstate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	mock_heartbeat "github.com/moira-alert/moira/mock/heartbeat"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
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
	defaultLocalCluster := moira.DefaultLocalCluster
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	mock := configureWorker(t, true)
	t.Run("SelfCheckWorker should call all heartbeats checks", func(t *testing.T) {
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil).Times(2)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil).Times(2)
		mock.database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), nil).Times(2)
		mock.database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(1), nil)

		// Start worker after configuring Mock to avoid race conditions
		err := mock.selfCheckWorker.Start()
		require.NoError(t, err)

		assert.Len(t, mock.selfCheckWorker.heartbeatsGraph[0], 1)
		assert.Len(t, mock.selfCheckWorker.heartbeatsGraph[1], 4)

		const oneTickDelay = time.Millisecond * 1500

		time.Sleep(oneTickDelay) // wait for one tick of worker

		err = mock.selfCheckWorker.Stop()
		require.NoError(t, err)
	})

	mock.mockCtrl.Finish()
}

func TestSelfCheckWorker_sendMessages(t *testing.T) {
	t.Run("Should call notifier send", func(t *testing.T) {
		mock := configureWorker(t, true)
		err := mock.selfCheckWorker.Start()
		require.NoError(t, err)

		mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

		var events []heartbeatNotificationEvent

		mock.selfCheckWorker.sendMessages(events)

		err = mock.selfCheckWorker.Stop()
		require.NoError(t, err)
		mock.mockCtrl.Finish()
	})

	t.Run("Should send user notifications if selfCheck state changes", func(t *testing.T) {
		cases := []struct {
			oldState               moira.SelfStateWorkerState
			state                  moira.SelfStateWorkerState
			isNotificationExpected bool
		}{
			{
				oldState:               moira.SelfStateWorkerOK,
				state:                  moira.SelfStateWorkerWARN,
				isNotificationExpected: false,
			},
			{
				// NOTE: Impossible case but need to check
				oldState:               moira.SelfStateWorkerOK,
				state:                  moira.SelfStateERROR,
				isNotificationExpected: false,
			},
			{
				oldState:               moira.SelfStateWorkerWARN,
				state:                  moira.SelfStateWorkerERROR,
				isNotificationExpected: true,
			},
			{
				oldState:               moira.SelfStateWorkerERROR,
				state:                  moira.SelfStateWorkerOK,
				isNotificationExpected: true,
			},
		}

		for _, testCase := range cases {
			t.Run(fmt.Sprintf("should send: %v, state: %v -> %v", testCase.isNotificationExpected, testCase.oldState, testCase.state), func(t *testing.T) {
				mock := configureWorker(t, true)
				err := mock.selfCheckWorker.Start()
				require.NoError(t, err)

				if testCase.isNotificationExpected {
					mock.database.EXPECT().GetTagsSubscriptions([]string{"tag"}).Return(nil, nil)
				}

				mock.selfCheckWorker.oldState = testCase.oldState
				mock.selfCheckWorker.state = testCase.state

				mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

				events := []heartbeatNotificationEvent{
					{
						NotificationEvent: moira.NotificationEvent{},
						CheckTags:         []string{"tag"},
					},
				}

				mock.selfCheckWorker.sendMessages(events)

				err = mock.selfCheckWorker.Stop()
				require.NoError(t, err)
				mock.mockCtrl.Finish()
			})
		}
	})
}

func TestSelfCheckWorker_handleGraphExecutionResult(t *testing.T) {
	t.Run("Should change own state in full cycle", func(t *testing.T) {
		mock := configureWorker(t, false)
		nowTS := time.Now()

		successGraphResult1 := graphExecutionResult{
			lastSuccessCheckElapsedTime: nowTS.Unix(),
			nowTimestamp:                time.Duration(nowTS.UnixNano()),
			hasErrors:                   false,
			needTurnOffNotifier:         false,
			errorMessages:               nil,
			checksTags:                  nil,
		}

		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil).Times(2)

		mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult1)

		assert.Empty(t, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.state)

		successGraphResult2 := successGraphResult1
		successGraphResult2.nowTimestamp = time.Duration(nowTS.UnixNano()) + 500*time.Millisecond

		events := mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult2)

		assert.Empty(t, events)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.state)

		errorGraphResult1 := graphExecutionResult{
			lastSuccessCheckElapsedTime: nowTS.Unix(),
			nowTimestamp:                time.Duration(nowTS.UnixNano()) + 1*time.Second,
			hasErrors:                   true,
			needTurnOffNotifier:         true,
			errorMessages:               []string{"some error"},
			checksTags:                  []string{"tag"},
		}

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult1)

		assert.Len(t, events, 1)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerWARN, mock.selfCheckWorker.state)

		errorGraphResult2 := errorGraphResult1
		errorGraphResult2.nowTimestamp = time.Duration(nowTS.UnixNano()) + 1500*time.Millisecond

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult2)

		assert.Empty(t, events)
		assert.Equal(t, moira.SelfStateWorkerWARN, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerWARN, mock.selfCheckWorker.state)

		errorGraphResult3 := errorGraphResult2
		errorGraphResult3.nowTimestamp = time.Duration(nowTS.UnixNano()) + 3000*time.Millisecond

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult3)

		assert.Len(t, events, 1)
		assert.Equal(t, moira.SelfStateWorkerWARN, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerERROR, mock.selfCheckWorker.state)

		errorGraphResult4 := errorGraphResult3
		errorGraphResult4.nowTimestamp = time.Duration(nowTS.UnixNano()) + 3500*time.Millisecond

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult4)

		assert.Empty(t, events)
		assert.Equal(t, moira.SelfStateWorkerERROR, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerERROR, mock.selfCheckWorker.state)

		successGraphResult3 := successGraphResult2
		successGraphResult3.nowTimestamp = time.Duration(nowTS.UnixNano()) + 4000*time.Millisecond

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateOK)
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateERROR,
		}, nil)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult3)

		assert.Len(t, events, 1)
		assert.Equal(t, moira.SelfStateWorkerERROR, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.state)

		successGraphResult4 := successGraphResult3
		successGraphResult4.nowTimestamp = time.Duration(nowTS.Unix()) + 4500*time.Millisecond

		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult4)

		assert.Empty(t, events)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.oldState)
		assert.Equal(t, moira.SelfStateWorkerOK, mock.selfCheckWorker.state)
	})
}

func TestSelfCheckWorker_constructUserNotification(t *testing.T) {
	t.Run("Should resemble events to contacts trought system tags", func(t *testing.T) {
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
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
		mock.mockCtrl.Finish()
	})
}

func TestSelfCheckWorker_enableNotifierIfNeed(t *testing.T) {
	mock := configureWorker(t, false)

	t.Run("Should enable if notifier is disabled by auto", func(t *testing.T) {
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateOK)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		require.NoError(t, err)
		assert.True(t, notifierEnabled)
	})

	t.Run("Should switch notifier to AUTO if enabled manually", func(t *testing.T) {
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}, nil)

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateOK)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		require.NoError(t, err)
		assert.True(t, notifierEnabled)
	})

	t.Run("Should not enable if notifier is disabled manually", func(t *testing.T) {
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorManual,
		}, nil)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		require.NoError(t, err)
		assert.False(t, notifierEnabled)
	})

	t.Run("Should not enable if notifier is disabled by a trigger", func(t *testing.T) {
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorTrigger,
		}, nil)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		require.NoError(t, err)
		assert.False(t, notifierEnabled)
	})

	t.Run("Should not enable notifier if it is already enabled", func(t *testing.T) {
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		require.NoError(t, err)
		assert.False(t, notifierEnabled)
	})

	t.Run("Should not enable notifier if getting state throw error", func(t *testing.T) {
		expected_err := fmt.Errorf("error")
		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{}, expected_err)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		assert.Equal(t, err, expected_err)
		assert.False(t, notifierEnabled)
	})

	t.Run("Should not enable notifier if notifier enabling returns error", func(t *testing.T) {
		expected_err := fmt.Errorf("error")

		mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorAutomatic,
		}, nil)

		mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateOK).Return(expected_err)

		notifierEnabled, err := mock.selfCheckWorker.enableNotifierIfPossible()
		assert.Equal(t, err, expected_err)
		assert.False(t, notifierEnabled)
	})
}

func TestSelfCheckWorker_Start(t *testing.T) {
	mock := configureWorker(t, false)
	t.Run("When Contact not corresponds to any Sender", func(t *testing.T) {
		mock.notif.EXPECT().GetSenders().Return(nil)

		t.Run("Start should return error", func(t *testing.T) {
			err := mock.selfCheckWorker.Start()
			require.Error(t, err)
		})
	})
}

func TestSelfCheckWorker(t *testing.T) {
	t.Run("Test checked heartbeat", func(t *testing.T) {
		err := errors.New("test error")
		now := time.Now().Unix()

		mock := configureWorker(t, false)

		t.Run("Test handle error and no needed send events", func(t *testing.T) {
			check := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{{check}}

			mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				Actor: moira.SelfStateActorAutomatic,
				State: moira.SelfStateOK,
			}, nil)

			check.EXPECT().Check(now).Return(int64(0), false, err)

			events := mock.selfCheckWorker.handleCheckServices(now)
			assert.Nil(t, events)
		})

		t.Run("Test turn off notification", func(t *testing.T) {
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
			mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR)

			events := mock.selfCheckWorker.handleCheckServices(now)
			assert.Len(t, events, 1)
		})

		t.Run("Test turn on notification", func(t *testing.T) {
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{
				{first},
				{second},
			}

			first.EXPECT().Check(now).Return(int64(15), false, nil)
			second.EXPECT().Check(now).Return(int64(15), false, nil)

			mock.database.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				State: moira.SelfStateERROR,
				Actor: moira.SelfStateActorAutomatic,
			}, nil)
			mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateOK)

			events := mock.selfCheckWorker.handleCheckServices(now)
			assert.Len(t, events, 1)
		})

		t.Run("Test of sending notifications from a check", func(t *testing.T) {
			now = time.Now().Unix()
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.heartbeatsGraph = heartbeatsGraph{
				{first},
				{second},
			}

			first.EXPECT().Check(now).Return(int64(0), true, nil)
			first.EXPECT().GetErrorMessage().Return(moira.SelfStateERROR)
			first.EXPECT().NeedTurnOffNotifier().Return(true)
			first.EXPECT().GetCheckTags().Return([]string{})
			mock.database.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorAutomatic, moira.SelfStateERROR).Return(err)
			mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

			mock.selfCheckWorker.check(now)
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
		UserNotificationsInterval:      2 * time.Second,
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
