package cmd

import (
	"testing"
	"time"

	"github.com/moira-alert/moira/database/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRedisConfig(t *testing.T) {
	Convey("Test RedisConfig.GetSettings", t, func() {
		Convey("With empty config", func() {
			redisCfg := RedisConfig{}

			expected := redis.DatabaseConfig{
				Addrs: []string{""},
			}
			databaseCfg := redisCfg.GetSettings()
			So(databaseCfg, ShouldResemble, expected)
		})

		Convey("With filled config", func() {
			redisCfg := RedisConfig{
				MasterName:       "test-master",
				Addrs:            "redis1:6379",
				SentinelUsername: "sentinel-user",
				SentinelPassword: "sentinel-pass",
				Username:         "user",
				Password:         "pass",
				MetricsTTL:       "1m",
				DialTimeout:      "1m",
				ReadTimeout:      "1m",
				WriteTimeout:     "1m",
				MaxRetries:       3,
			}

			expected := redis.DatabaseConfig{
				MasterName:       "test-master",
				Addrs:            []string{"redis1:6379"},
				SentinelUsername: "sentinel-user",
				SentinelPassword: "sentinel-pass",
				Username:         "user",
				Password:         "pass",
				MetricsTTL:       time.Minute,
				DialTimeout:      time.Minute,
				ReadTimeout:      time.Minute,
				WriteTimeout:     time.Minute,
				MaxRetries:       3,
			}
			databaseCfg := redisCfg.GetSettings()
			So(databaseCfg, ShouldResemble, expected)
		})
	})
}
