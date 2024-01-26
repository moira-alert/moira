package main

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"

	"github.com/moira-alert/moira/api"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_apiConfig_getSettings(t *testing.T) {
	Convey("Settings successfully filled", t, func() {
		metricTTLs := map[moira.ClusterKey]time.Duration{
			moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster):  time.Hour,
			moira.MakeClusterKey(moira.GraphiteRemote, moira.DefaultCluster): 24 * time.Hour,
		}

		apiConf := apiConfig{
			Listen:     "0000",
			EnableCORS: true,
		}

		expectedResult := &api.Config{
			EnableCORS: true,
			Listen:     "0000",
			MetricsTTL: metricTTLs,
			Flags:      api.FeatureFlags{IsReadonlyEnabled: true},
		}

		result := apiConf.getSettings(metricTTLs, api.FeatureFlags{IsReadonlyEnabled: true})
		So(result, ShouldResemble, expectedResult)
	})
}

func Test_webConfig_getFeatureFlags(t *testing.T) {
	Convey("Flags successfully filled", t, func() {
		webConf := webConfig{
			FeatureFlags: featureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
			},
		}

		expectedResult := api.FeatureFlags{
			IsPlottingDefaultOn:              true,
			IsPlottingAvailable:              true,
			IsSubscriptionToAllTagsAvailable: true,
		}

		result := webConf.getFeatureFlags()
		So(result, ShouldResemble, expectedResult)
	})
}

func Test_webConfig_getDefault(t *testing.T) {
	Convey("Flags successfully filled", t, func() {
		expectedResult := config{
			Redis: cmd.RedisConfig{
				Addrs:       "localhost:6379",
				MetricsTTL:  "1h",
				DialTimeout: "500ms",
				MaxRetries:  3,
			},
			Logger: cmd.LoggerConfig{
				LogFile:         "stdout",
				LogLevel:        "info",
				LogPrettyFormat: false,
			},
			API: apiConfig{
				Listen:     ":8081",
				EnableCORS: false,
			},
			Web: webConfig{
				RemoteAllowed: false,
				FeatureFlags: featureFlags{
					IsPlottingDefaultOn:              true,
					IsPlottingAvailable:              true,
					IsSubscriptionToAllTagsAvailable: true,
				},
			},
			Telemetry: cmd.TelemetryConfig{
				Listen: ":8091",
				Graphite: cmd.GraphiteConfig{
					Enabled:      false,
					RuntimeStats: false,
					URI:          "localhost:2003",
					Prefix:       "DevOps.Moira",
					Interval:     "60s",
				},
				Pprof: cmd.ProfilerConfig{Enabled: false},
			},
			Remotes: cmd.RemotesConfig{},
			NotificationHistory: cmd.NotificationHistoryConfig{
				NotificationHistoryTTL:        "48h",
				NotificationHistoryQueryLimit: -1,
			},
		}

		result := getDefault()
		So(result, ShouldResemble, expectedResult)
	})
}

func Test_webConfig_getSettings(t *testing.T) {
	Convey("Empty config, fill it", t, func() {
		config := webConfig{}

		settings := config.getSettings(true)
		So(settings, ShouldResemble, &api.WebConfig{
			RemoteAllowed: true,
			Contacts:      []api.WebContact{},
		})
	})

	Convey("Default config, fill it", t, func() {
		config := getDefault()

		settings := config.Web.getSettings(true)
		So(settings, ShouldResemble, &api.WebConfig{
			RemoteAllowed: true,
			Contacts:      []api.WebContact{},
			FeatureFlags: api.FeatureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
			},
		})
	})

	Convey("Not empty config, fill it", t, func() {
		config := webConfig{
			SupportEmail:  "lalal@mail.la",
			RemoteAllowed: false,
			Contacts: []webContact{
				{
					ContactType:     "slack",
					ContactLabel:    "label",
					ValidationRegex: "t(\\d+)",
					Placeholder:     "",
					Help:            "help",
				},
			},
			FeatureFlags: featureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
				IsReadonlyEnabled:                false,
			},
			Sentry: sentryConfig{
				DSN: "test dsn",
			},
		}

		settings := config.getSettings(true)
		So(settings, ShouldResemble, &api.WebConfig{
			SupportEmail:  "lalal@mail.la",
			RemoteAllowed: true,
			Contacts: []api.WebContact{
				{
					ContactType:     "slack",
					ContactLabel:    "label",
					ValidationRegex: "t(\\d+)",
					Help:            "help",
				},
			},
			FeatureFlags: api.FeatureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
				IsReadonlyEnabled:                false,
			},
			Sentry: api.Sentry{
				DSN: "test dsn",
			},
		})
	})
}
