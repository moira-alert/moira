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
			moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster): time.Hour,
			moira.DefaultGraphiteRemoteCluster:                              24 * time.Hour,
		}

		webConfig := &webConfig{
			ContactsTemplate: []webContact{
				{
					ContactType: "test",
				},
			},
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
			Authorization: api.Authorization{
				AdminList: make(map[string]struct{}),
				AllowedContactTypes: map[string]struct{}{
					"test": {},
				},
			},
		}

		result := apiConf.getSettings(metricTTLs, api.FeatureFlags{IsReadonlyEnabled: true}, webConfig)
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
				Addrs:           "localhost:6379",
				MetricsTTL:      "1h",
				MaxRetries:      3,
				MaxRedirects:    3,
				DialTimeout:     "500ms",
				ReadTimeout:     "3s",
				WriteTimeout:    "3s",
				PoolTimeout:     "4s",
				PoolSize:        0,
				PoolSizePerProc: 5,
			},
			Logger: cmd.LoggerConfig{
				LogFile:         "stdout",
				LogLevel:        "info",
				LogPrettyFormat: false,
			},
			API: apiConfig{
				Listen:     ":8081",
				EnableCORS: false,
				Limits: LimitsConfig{
					Pager: PagerLimits{
						TTL: api.DefaultTriggerPagerTTL,
					},
					Trigger: TriggerLimitsConfig{
						MaxNameSize: api.DefaultTriggerNameMaxSize,
					},
					Team: TeamLimitsConfig{
						MaxNameSize:        api.DefaultTeamNameMaxSize,
						MaxDescriptionSize: api.DefaultTeamDescriptionMaxSize,
					},
				},
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
				NotificationHistoryTTL: "48h",
			},
		}

		result := getDefault()
		So(result, ShouldResemble, expectedResult)
	})
}

func Test_webConfig_getSettings(t *testing.T) {
	metricSourceClustersDefault := []api.MetricSourceCluster{{
		TriggerSource: moira.GraphiteLocal,
		ClusterId:     moira.DefaultCluster,
		ClusterName:   "Graphite Local",
	}}
	remotesDefault := cmd.RemotesConfig{}

	Convey("Empty config, fill it", t, func() {
		config := webConfig{}

		settings := config.getSettings(true, remotesDefault)
		So(settings, ShouldResemble, &api.WebConfig{
			RemoteAllowed:        true,
			Contacts:             []api.WebContact{},
			MetricSourceClusters: metricSourceClustersDefault,
		})
	})

	Convey("Default config, fill it", t, func() {
		config := getDefault()

		settings := config.Web.getSettings(true, remotesDefault)
		So(settings, ShouldResemble, &api.WebConfig{
			RemoteAllowed: true,
			Contacts:      []api.WebContact{},
			FeatureFlags: api.FeatureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
			},
			MetricSourceClusters: metricSourceClustersDefault,
		})
	})

	Convey("Not empty config, fill it", t, func() {
		config := webConfig{
			SupportEmail:  "lalal@mail.la",
			RemoteAllowed: false,
			ContactsTemplate: []webContact{
				{
					ContactType:     "slack",
					ContactLabel:    "label",
					LogoURI:         "/test/test.svg",
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
				DSN:      "test dsn",
				Platform: "dev",
			},
		}

		settings := config.getSettings(true, remotesDefault)
		So(settings, ShouldResemble, &api.WebConfig{
			SupportEmail:  "lalal@mail.la",
			RemoteAllowed: true,
			Contacts: []api.WebContact{
				{
					ContactType:     "slack",
					ContactLabel:    "label",
					LogoURI:         "/test/test.svg",
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
				DSN:      "test dsn",
				Platform: "dev",
			},
			MetricSourceClusters: metricSourceClustersDefault,
		})
	})

	Convey("Empty config, non default cluster list", t, func() {
		config := webConfig{}
		remotes := cmd.RemotesConfig{
			Graphite: []cmd.GraphiteRemoteConfig{{
				RemoteCommonConfig: cmd.RemoteCommonConfig{
					ClusterId:   "default",
					ClusterName: "Graphite Remote 123",
				},
			}},
			Prometheus: []cmd.PrometheusRemoteConfig{{
				RemoteCommonConfig: cmd.RemoteCommonConfig{
					ClusterId:   "default",
					ClusterName: "Prometheus Remote 888",
				},
			}},
		}

		settings := config.getSettings(true, remotes)
		So(settings, ShouldResemble, &api.WebConfig{
			RemoteAllowed: true,
			Contacts:      []api.WebContact{},
			MetricSourceClusters: []api.MetricSourceCluster{
				{
					TriggerSource: moira.GraphiteLocal,
					ClusterId:     moira.DefaultCluster,
					ClusterName:   "Graphite Local",
				},
				{
					TriggerSource: moira.GraphiteRemote,
					ClusterId:     moira.DefaultCluster,
					ClusterName:   "Graphite Remote 123",
				},
				{
					TriggerSource: moira.PrometheusRemote,
					ClusterId:     moira.DefaultCluster,
					ClusterName:   "Prometheus Remote 888",
				},
			},
		})
	})
}

func Test_webConfig_getCelebrationMode(t *testing.T) {
	Convey("Available celebration mode, should return mode", t, func() {
		celebrationMode := getCelebrationMode("new_year")
		So(celebrationMode, ShouldEqual, api.CelebrationMode("new_year"))
	})

	Convey("Not available celebration mode, should return empty string", t, func() {
		celebrationMode := getCelebrationMode("blablabla")
		So(celebrationMode, ShouldEqual, "")
	})
}

func Test_webConfig_validate(t *testing.T) {
	Convey("With empty web config", t, func() {
		config := webConfig{}

		err := config.validate()
		So(err, ShouldBeNil)
	})

	Convey("With invalid contact template pattern", t, func() {
		config := webConfig{
			ContactsTemplate: []webContact{
				{
					ValidationRegex: "**",
				},
			},
		}

		err := config.validate()
		So(err, ShouldNotBeNil)
	})

	Convey("With valid contact template pattern", t, func() {
		config := webConfig{
			ContactsTemplate: []webContact{
				{
					ValidationRegex: ".*",
				},
			},
		}

		err := config.validate()
		So(err, ShouldBeNil)
	})
}
