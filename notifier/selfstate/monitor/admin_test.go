package monitor

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var defaultHeartbeatersConfig = heartbeat.HeartbeatersConfig{
	DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
		HeartbeaterBaseConfig: heartbeat.HeartbeaterBaseConfig{
			Enabled: true,
		},
		RedisDisconnectDelay: 1,
	},
	FilterCfg:        heartbeat.FilterHeartbeaterConfig{},
	LocalCheckerCfg:  heartbeat.LocalCheckerHeartbeaterConfig{},
	RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{},
	NotifierCfg:      heartbeat.NotifierHeartbeaterConfig{},
}

func TestValidateConfig(t *testing.T) {
	senders := map[string]bool{
		"telegram": true,
	}

	validationErr := validator.ValidationErrors{}

	Convey("Test Validate", t, func() {
		Convey("With disabled admin and user selfchecks", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled: false,
				},
			}

			err := cfg.validate(senders)
			So(err, ShouldBeNil)
		})

		Convey("Without heartbeats config", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled: true,
				},
			}

			err := cfg.validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin notice interval", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
				},
			}

			err := cfg.validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin check interval", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
				},
			}

			err := cfg.validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin contacts", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
					CheckInterval:   time.Minute,
				},
			}

			err := cfg.validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("With empty admin contacts", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
					CheckInterval:   time.Minute,
				},
				AdminContacts: []map[string]string{},
			}

			err := cfg.validate(senders)
			So(err, ShouldBeNil)
		})

		Convey("With unknown admin contact type", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
					CheckInterval:   time.Minute,
				},
				AdminContacts: []map[string]string{
					{
						"type": "test-contact-type",
					},
				},
			}

			err := cfg.validate(senders)
			So(err, ShouldResemble, fmt.Errorf("unknown contact type in admin config: [%s]", cfg.AdminContacts[0]["type"]))
		})

		Convey("Without admin contact value", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
					CheckInterval:   time.Minute,
				},
				AdminContacts: []map[string]string{
					{
						"type": "telegram",
					},
				},
			}

			err := cfg.validate(senders)
			So(err, ShouldResemble, fmt.Errorf("value for [%s] must be present", cfg.AdminContacts[0]["type"]))
		})

		Convey("With valid admin contact type and value", func() {
			cfg := AdminMonitorConfig{
				MonitorBaseConfig: MonitorBaseConfig{
					Enabled:         true,
					HeartbeatersCfg: defaultHeartbeatersConfig,
					NoticeInterval:  time.Minute,
					CheckInterval:   time.Minute,
				},
				AdminContacts: []map[string]string{
					{
						"type":  "telegram",
						"value": "@webcamsmodel",
					},
				},
			}

			err := cfg.validate(senders)
			So(err, ShouldBeNil)
		})
	})
}

func TestAdminSendNotifications(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockNotifier := mock_notifier.NewMockNotifier(mockCtrl)

	adminCfg := AdminMonitorConfig{
		AdminContacts: []map[string]string{
			{
				"type":  "telegram",
				"value": "@webcamsmodel",
			},
		},
	}

	sendingWG := &sync.WaitGroup{}

	adminMonitor := adminMonitor{
		notifier: mockNotifier,
		adminCfg: adminCfg,
	}

	Convey("Test sendNotifications", t, func() {
		Convey("With empty notification packages", func() {
			pkgs := []notifier.NotificationPackage{}
			err := adminMonitor.sendNotifications(pkgs)
			So(err, ShouldBeNil)
		})

		Convey("With correct sending notification packages", func() {
			pkgs := []notifier.NotificationPackage{
				{},
			}
			pkgWithContact := &notifier.NotificationPackage{
				Contact: moira.ContactData{
					Type:  adminCfg.AdminContacts[0]["type"],
					Value: adminCfg.AdminContacts[0]["value"],
				},
			}
			mockNotifier.EXPECT().Send(pkgWithContact, sendingWG).Times(1)
			err := adminMonitor.sendNotifications(pkgs)
			So(err, ShouldBeNil)
		})
	})
}
