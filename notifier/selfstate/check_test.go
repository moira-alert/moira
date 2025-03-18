package selfstate

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	notifier2 "github.com/moira-alert/moira/notifier"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestSelfCheckWorker_check(t *testing.T) {
	Convey("Check should", t, func() {
		Convey("Send notifications to admins and users on single heartbeat error", func() {
			var timeDelta int64 = 100
			nowTS := time.Now().Unix() + timeDelta
			worker := createWorker(t)

			fillDatabase(worker.database)
			worker.database.EXPECT().GetChecksUpdatesCount().Return(int64(0), fmt.Errorf(""))

			val := float64(timeDelta)
			var sendingWG sync.WaitGroup

			for _, contact := range worker.conf.Contacts {
				toAdminPack := notifier2.NotificationPackage{
					Trigger: moira.TriggerData{
						Name:       "Moira health check",
						ErrorValue: float64(0),
					},
					Contact: moira.ContactData{
						Type:  contact["type"],
						Value: contact["value"],
					},
					DontResend: true,
					Events: []moira.NotificationEvent{
						{
							Timestamp: nowTS,
							OldState:  moira.StateNODATA,
							State:     moira.StateERROR,
							Metric:    "Redis disconnected",
							Value:     &val,
						},
					},
				}

				worker.notif.EXPECT().Send(&toAdminPack, &sendingWG).Times(1)
			}

			toUsersPack := notifier2.NotificationPackage{
				Trigger: moira.TriggerData{
					Name:       "Moira health check",
					ErrorValue: float64(0),
				},
				Contact: moira.ContactData{
					ID:    "contact1",
					Type:  "user-mail",
					Value: "user@userdomain.com",
				},
				DontResend: true,
				Events: []moira.NotificationEvent{
					{
						Timestamp: nowTS,
						OldState:  moira.StateNODATA,
						State:     moira.StateERROR,
						Metric:    "Redis disconnected",
						Value:     &val,
					},
				},
			}

			worker.notif.EXPECT().Send(&toUsersPack, &sendingWG).Times(1)

			worker.selfCheckWorker.check(nowTS, 10)
		})
	})
}

func createWorker(t *testing.T) *selfCheckWorkerMock {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}
	conf := Config{
		Enabled: true,
		Contacts: []map[string]string{
			adminContact,
		},
		RedisDisconnectDelaySeconds:    1,
		LastMetricReceivedDelaySeconds: 60,
		LastCheckDelaySeconds:          120,
		NoticeIntervalSeconds:          60,
		LastRemoteCheckDelaySeconds:    120,
		CheckInterval:                  1 * time.Second,
		Checks: ChecksConfig{
			Database: HeartbeatConfig{
				SystemTags: []string{"sys-tag-database", "moira-fatal"},
			},
			Filter: HeartbeatConfig{
				SystemTags: []string{"sys-tag-filter", "moira-fatal"},
			},
			LocalChecker: HeartbeatConfig{
				SystemTags: []string{"sys-tag-local-checker"},
			},
			RemoteChecker: HeartbeatConfig{
				SystemTags: []string{"sys-tag-remote-checker", "moira-fatal"},
			},
			Notifier: HeartbeatConfig{
				SystemTags: []string{"sys-tag-notifier", "moira-fatal"},
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	return &selfCheckWorkerMock{
		selfCheckWorker: NewSelfCheckWorker(logger, database, notif, conf),
		database:        database,
		notif:           notif,
		conf:            conf,
		mockCtrl:        mockCtrl,
	}
}

func fillDatabase(database *mock_moira_alert.MockDatabase) {
	contacts := []*moira.ContactData{
		{
			ID:    "contact1",
			Type:  "user-mail",
			Value: "user@userdomain.com",
		},
	}
	database.EXPECT().GetContacts(moira.Map(contacts, func(c *moira.ContactData) string { return c.ID })).Return(contacts, nil)
	database.EXPECT().SetNotifierState(gomock.Any()).Return(nil)
	database.EXPECT().GetTagsSubscriptions([]string{"sys-tag-database", "moira-fatal"}).Return([]*moira.SubscriptionData{
		{
			Contacts: []string{"contact1"},
			Tags:     []string{"sys-tag-database", "moira-fatal"},
			Enabled:  true,
		},
	}, nil)
}
