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

	Convey("Should send user notifications if selfCheck state changes", t, func() {
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
			Convey(fmt.Sprintf("should send: %v, state: %v -> %v", testCase.isNotificationExpected, testCase.oldState, testCase.state), func() {
				mock := configureWorker(t, true)
				err := mock.selfCheckWorker.Start()
				So(err, ShouldBeNil)

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
				So(err, ShouldBeNil)
				mock.mockCtrl.Finish()
			})
		}
	})
}

func TestSelfCheckWorker_handleGraphExecutionResult(t *testing.T) {
	Convey("Should change own state in full cycle", t, func() {
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

		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil).Times(2)

		mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult1)

		So(mock.selfCheckWorker.oldState, ShouldEqual, "")
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerOK)

		successGraphResult2 := successGraphResult1
		successGraphResult2.nowTimestamp = time.Duration(nowTS.UnixNano()) + 500*time.Millisecond

		events := mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult2)

		So(len(events), ShouldEqual, 0)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerOK)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerOK)

		errorGraphResult1 := graphExecutionResult{
			lastSuccessCheckElapsedTime: nowTS.Unix(),
			nowTimestamp:                time.Duration(nowTS.UnixNano()) + 1*time.Second,
			hasErrors:                   true,
			needTurnOffNotifier:         true,
			errorMessages:               []string{"some error"},
			checksTags:                  []string{"tag"},
		}

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult1)

		So(len(events), ShouldEqual, 1)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerOK)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerWARN)

		errorGraphResult2 := errorGraphResult1
		errorGraphResult2.nowTimestamp = time.Duration(nowTS.UnixNano()) + 1500*time.Millisecond

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult2)

		So(len(events), ShouldEqual, 0)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerWARN)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerWARN)

		errorGraphResult3 := errorGraphResult2
		errorGraphResult3.nowTimestamp = time.Duration(nowTS.UnixNano()) + 3000*time.Millisecond

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult3)

		So(len(events), ShouldEqual, 1)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerWARN)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerERROR)

		errorGraphResult4 := errorGraphResult3
		errorGraphResult4.nowTimestamp = time.Duration(nowTS.UnixNano()) + 3500*time.Millisecond

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), errorGraphResult4)

		So(len(events), ShouldEqual, 0)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerERROR)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerERROR)

		successGraphResult3 := successGraphResult2
		successGraphResult3.nowTimestamp = time.Duration(nowTS.UnixNano()) + 4000*time.Millisecond

		mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateOK)
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateERROR,
		}, nil)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult3)

		So(len(events), ShouldEqual, 1)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerERROR)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerOK)

		successGraphResult4 := successGraphResult3
		successGraphResult4.nowTimestamp = time.Duration(nowTS.Unix()) + 4500*time.Millisecond

		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			Actor: moira.SelfStateActorAutomatic,
			State: moira.SelfStateOK,
		}, nil)

		events = mock.selfCheckWorker.handleGraphExecutionResult(nowTS.Unix(), successGraphResult4)

		So(len(events), ShouldEqual, 0)
		So(mock.selfCheckWorker.oldState, ShouldEqual, moira.SelfStateWorkerOK)
		So(mock.selfCheckWorker.state, ShouldEqual, moira.SelfStateWorkerOK)
	})
}

func TestSelfCheckWorker_constructUserNotification(t *testing.T) {
	Convey("Should resemble events to contacts trought system tags", t, func() {
		baseContact := moira.ContactData{
			ID:    "some-contact",
			Type:  "my_type",
			Value: "123",
		}

		baseSystemSubscription1 := moira.SubscriptionData{
			ID:       "sys-sub-1",
			Contacts: []string{baseContact.ID},
			Tags:     []string{"sys-tag1"},
		}
		baseSystemSubscription2 := moira.SubscriptionData{
			ID:       "sys-sub-2",
			Contacts: []string{baseContact.ID},
			Tags:     []string{"sys-tag2", "sys-tag-common"},
		}

		Convey("if owner does not exists", func() {
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
					Contact: baseContact,
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

			database.EXPECT().GetTagsSubscriptions(baseSystemSubscription1.Tags).Return([]*moira.SubscriptionData{
				&baseSystemSubscription1,
			}, nil)
			database.EXPECT().GetTagsSubscriptions(baseSystemSubscription2.Tags).Return([]*moira.SubscriptionData{
				&baseSystemSubscription2,
			}, nil)

			database.EXPECT().GetContacts([]string{baseContact.ID}).Return([]*moira.ContactData{
				&baseContact,
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

		Convey("if owner is user", func() {
			user := "user-1"

			contact := baseContact
			contact.User = user

			systemSubscription1 := baseSystemSubscription1
			systemSubscription1.User = user

			systemSubscription2 := baseSystemSubscription2
			systemSubscription2.User = user

			subscription1 := moira.SubscriptionData{
				ID:       "sub-1",
				Contacts: []string{contact.ID},
				Tags:     []string{"tag1", "tag2"},
				User:     user,
			}

			lastCheckData := moira.CheckData{
				State: moira.StateERROR,
			}

			notifAndTags := []heartbeatNotificationEvent{
				{
					NotificationEvent: moira.NotificationEvent{
						Metric: "Check passed!",
					},
					CheckTags: heartbeat.CheckTags{
						"sys-tag1",
					},
					NotifyAboutEnabledNotifier: true,
				},
				{
					NotificationEvent: moira.NotificationEvent{
						Metric: "Some another check passed!",
					},
					CheckTags: heartbeat.CheckTags{
						"sys-tag2", "sys-tag-common",
					},
					NotifyAboutEnabledNotifier: true,
				},
			}

			expected := []*notifier.NotificationPackage{
				{
					Contact: contact,
					Trigger: moira.TriggerData{
						Name:       "Moira health check",
						Desc:       "These triggers are in a bad state. Check them by tags:\n- [tag1|tag2](https://moira/?onlyProblems=true&tags%5B0%5D=tag1&tags%5B1%5D=tag2)\n",
						ErrorValue: float64(0),
					},
					Events: []moira.NotificationEvent{
						{
							Metric: "Check passed!",
						},
						{
							Metric: "Some another check passed!",
						},
					},
					DontResend: true,
				},
			}

			mockCtrl := gomock.NewController(t)
			database := mock_moira_alert.NewMockDatabase(mockCtrl)

			database.EXPECT().GetTagsSubscriptions(systemSubscription1.Tags).Return([]*moira.SubscriptionData{
				&systemSubscription1,
			}, nil)
			database.EXPECT().GetTagsSubscriptions(systemSubscription2.Tags).Return([]*moira.SubscriptionData{
				&systemSubscription2,
			}, nil)

			database.EXPECT().GetSubscription(systemSubscription1.ID).Return(systemSubscription1, nil).Times(1)
			database.EXPECT().GetSubscription(systemSubscription2.ID).Return(systemSubscription2, nil).Times(1)
			database.EXPECT().GetSubscription(subscription1.ID).Return(subscription1, nil).Times(1)

			for _, tag := range subscription1.Tags {
				database.EXPECT().GetTagTriggerIDs(tag).Return([]string{"trigger-1"}, nil).Times(1)
			}

			database.EXPECT().GetTriggerLastCheck("trigger-1").Return(lastCheckData, nil).Times(1)

			database.EXPECT().GetContacts([]string{contact.ID}).Return([]*moira.ContactData{
				&contact,
			}, nil).Times(2)

			database.EXPECT().GetUserSubscriptionIDs(user).Return([]string{
				systemSubscription1.ID,
				systemSubscription2.ID,
				subscription1.ID,
			}, nil).Times(1)

			logger, _ := logging.GetLogger("SelfState")
			notif := mock_notifier.NewMockNotifier(mockCtrl)

			mock := &selfCheckWorkerMock{
				selfCheckWorker: NewSelfCheckWorker(logger, database, notif, Config{
					FrontURL: "https://moira/",
					Checks: ChecksConfig{
						Filter: HeartbeatConfig{
							SystemTags: []string{"sys-tag1", "sys-tag2", "sys-tag-common"},
						},
					},
				}),
				mockCtrl: mockCtrl,
			}

			actual, err := mock.selfCheckWorker.constructUserNotification(notifAndTags)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			mock.mockCtrl.Finish()
		})
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

	Convey("Should not enable if notifier is disabled by a trigger", t, func() {
		mock.database.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateERROR,
			Actor: moira.SelfStateActorTrigger,
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

func TestSelfCheck_should_construct_links_to_triggers(t *testing.T) {
	mock := configureWorker(t, false)

	user := "user"
	systemTags := []string{"sys-tag1", "sys-tag2"}

	systemSubscription := moira.SubscriptionData{
		ID:   "sub-1",
		Tags: systemTags,
		User: user,
	}

	userSubscription1 := moira.SubscriptionData{
		ID:   "sub-2",
		Tags: []string{"tag1", "tag2"},
		User: user,
	}

	userSubscription2 := moira.SubscriptionData{
		ID:   "sub-3",
		Tags: []string{"tag3", "tag4"},
		User: user,
	}

	userSubscription3 := moira.SubscriptionData{
		ID:   "sub-4",
		Tags: []string{"tag5"},
		User: user,
	}

	mock.database.EXPECT().GetUserSubscriptionIDs(user).Return([]string{systemSubscription.ID, userSubscription1.ID, userSubscription2.ID, userSubscription3.ID}, nil)
	mock.database.EXPECT().GetSubscription(systemSubscription.ID).Return(systemSubscription, nil)
	mock.database.EXPECT().GetSubscription(userSubscription1.ID).Return(userSubscription1, nil)
	mock.database.EXPECT().GetSubscription(userSubscription2.ID).Return(userSubscription2, nil)
	mock.database.EXPECT().GetSubscription(userSubscription3.ID).Return(userSubscription3, nil)

	for _, tag := range userSubscription1.Tags {
		mock.database.EXPECT().GetTagTriggerIDs(tag).Return([]string{"trigger-1"}, nil)
	}

	for _, tag := range userSubscription2.Tags {
		mock.database.EXPECT().GetTagTriggerIDs(tag).Return([]string{"trigger-1"}, nil)
	}

	for _, tag := range userSubscription3.Tags {
		mock.database.EXPECT().GetTagTriggerIDs(tag).Return([]string{"trigger-2"}, nil)
	}

	mock.database.EXPECT().GetTriggerLastCheck("trigger-1").Return(moira.CheckData{
		State: moira.StateERROR,
	}, nil).Times(2)
	mock.database.EXPECT().GetTriggerLastCheck("trigger-2").Return(moira.CheckData{
		State: moira.StateOK,
	}, nil).Times(1)

	res, err := mock.selfCheckWorker.constructTriggersTable(&systemSubscription, systemTags)
	if err != nil {
		t.Fatalf("error not nil: %v", err)
	}

	expectedTable := []string{
		mock.conf.FrontURL + "?onlyProblems=true&tags%5B0%5D=tag1&tags%5B1%5D=tag2",
		mock.conf.FrontURL + "?onlyProblems=true&tags%5B0%5D=tag3&tags%5B1%5D=tag4",
	}
	actual := moira.Map(res, func(elem triggersTableElem) string { return elem.Link })

	if len(moira.SymmetricDiff(actual, expectedTable)) > 0 {
		t.Fatalf("trigger table invalid: %v", res)
	}
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
			So(len(events), ShouldEqual, 1)
		})

		Convey("Test of sending notifications from a check", func() {
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
			mock.database.EXPECT().SetNotifierState(moira.SelfStateActorAutomatic, moira.SelfStateERROR).Return(err)
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
		FrontURL:                       "https://moira-testing/",
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
