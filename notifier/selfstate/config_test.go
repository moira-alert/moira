package selfstate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
)

var defaultHeartbeatersConfig = HeartbeatsCfg{
	DatabaseCfg: heartbeat.DatabaseHeartbeaterConfig{
		RedisDisconnectDelay: time.Minute,
	},
	FilterCfg: heartbeat.FilterHeartbeaterConfig{
		MetricReceivedDelay: time.Minute,
	},
	LocalCheckerCfg: heartbeat.LocalCheckerHeartbeaterConfig{
		LocalCheckDelay: time.Minute,
	},
	RemoteCheckerCfg: heartbeat.RemoteCheckerHeartbeaterConfig{
		RemoteCheckDelay: time.Minute,
	},
	NotifierCfg: heartbeat.NotifierHeartbeaterConfig{},
}

func TestValidateConfig(t *testing.T) {
	senders := map[string]bool{
		"telegram": true,
	}

	validationErr := validator.ValidationErrors{}

	Convey("Test Validate", t, func() {
		Convey("With disabled admin and user selfchecks", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(err, ShouldBeNil)
		})

		Convey("Without heartbeats config", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: true,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin notice interval", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:       true,
							HeartbeatsCfg: defaultHeartbeatersConfig,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without user notice interval", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:       true,
							HeartbeatsCfg: defaultHeartbeatersConfig,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin check interval", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without user check interval", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("Without admin contacts", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
							CheckInterval:  time.Minute,
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("With empty admin contacts", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
							CheckInterval:  time.Minute,
						},
						AdminContacts: []map[string]string{},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.As(err, &validationErr), ShouldBeTrue)
		})

		Convey("With unknown contact type", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
							CheckInterval:  time.Minute,
						},
						AdminContacts: []map[string]string{
							{
								"type": "test-contact-type",
							},
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.Unwrap(err), ShouldResemble, fmt.Errorf("unknown contact type [%s]", cfg.Monitor.AdminCfg.AdminContacts[0]["type"]))
		})

		Convey("Without contact value", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
							CheckInterval:  time.Minute,
						},
						AdminContacts: []map[string]string{
							{
								"type": "telegram",
							},
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(errors.Unwrap(err), ShouldResemble, fmt.Errorf("value for [%s] must be present", cfg.Monitor.AdminCfg.AdminContacts[0]["type"]))
		})

		Convey("With valid contact type and value", func() {
			cfg := Config{
				Monitor: MonitorConfig{
					AdminCfg: AdminMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled:        true,
							HeartbeatsCfg:  defaultHeartbeatersConfig,
							NoticeInterval: time.Minute,
							CheckInterval:  time.Minute,
						},
						AdminContacts: []map[string]string{
							{
								"type":  "telegram",
								"value": "@webcamsmodel",
							},
						},
					},
					UserCfg: UserMonitorConfig{
						MonitorBaseConfig: MonitorBaseConfig{
							Enabled: false,
						},
					},
				},
			}

			err := cfg.Validate(senders)
			So(err, ShouldBeNil)
		})
	})
}
