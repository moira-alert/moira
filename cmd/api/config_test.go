package main

import (
	"testing"
	"time"

	"github.com/moira-alert/moira/cmd"

	"github.com/moira-alert/moira/api"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_apiConfig_getSettings(t *testing.T) {
	Convey("Settings successfully filled", t, func() {
		apiConf := apiConfig{
			Listen:     "0000",
			EnableCORS: true,
		}

		expectedResult := &api.Config{
			EnableCORS:              true,
			Listen:                  "0000",
			GraphiteLocalMetricTTL:  time.Hour,
			GraphiteRemoteMetricTTL: 24 * time.Hour,
			Flags:                   api.FeatureFlags{IsReadonlyEnabled: true},
		}

		result := apiConf.getSettings("1h", "24h", api.FeatureFlags{IsReadonlyEnabled: true})
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
			Remote: cmd.RemoteConfig{
				Timeout:    "60s",
				MetricsTTL: "7d",
			},
			Prometheus: cmd.PrometheusConfig{
				Timeout:      "60s",
				MetricsTTL:   "7d",
				Retries:      1,
				RetryTimeout: "10s",
			},
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
		wC := webConfig{}

		result, err := wC.getSettings(true)
		So(err, ShouldBeEmpty)
		So(string(result), ShouldResemble, `{"remoteAllowed":true,"contacts":[],"featureFlags":{"isPlottingDefaultOn":false,"isPlottingAvailable":false,"isSubscriptionToAllTagsAvailable":false,"isReadonlyEnabled":false}}`)
	})

	Convey("Default config, fill it", t, func() {
		config := getDefault()

		result, err := config.Web.getSettings(true)
		So(err, ShouldBeEmpty)
		So(string(result), ShouldResemble, `{"remoteAllowed":true,"contacts":[],"featureFlags":{"isPlottingDefaultOn":true,"isPlottingAvailable":true,"isSubscriptionToAllTagsAvailable":true,"isReadonlyEnabled":false}}`)
	})

	Convey("Not empty config, fill it", t, func() {
		wC := webConfig{
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
				IsPlottingAvailable:              false,
				IsSubscriptionToAllTagsAvailable: true,
			},
		}

		result, err := wC.getSettings(true)
		So(err, ShouldBeEmpty)
		So(string(result), ShouldResemble, `{"supportEmail":"lalal@mail.la","remoteAllowed":true,"contacts":[{"type":"slack","label":"label","validation":"t(\\d+)","help":"help"}],"featureFlags":{"isPlottingDefaultOn":true,"isPlottingAvailable":false,"isSubscriptionToAllTagsAvailable":true,"isReadonlyEnabled":false}}`)
	})
}
