package main

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func makeDb() (moira.Logger, *redis.DbConnector) {
	logger, err := logging.ConfigureLog("stdout", "debug", "test", true)
	if err != nil {
		panic("Failed to init logger " + err.Error())
	}

	database := redis.NewDatabase(logger, redis.DatabaseConfig{
		Addrs:         []string{"localhost:6370", "localhost:6371", "localhost:6372", "localhost:6373", "localhost:6374", "localhost:6375"},
		MetricsTTL:    time.Hour * 3,
		DialTimeout:   time.Second * 3,
		ReadTimeout:   time.Second * 3,
		WriteTimeout:  time.Second * 3,
		MaxRetries:    10,
		ReadOnly:      true,
		RouteRandomly: true,
	}, redis.NotificationHistoryConfig{}, redis.NotificationConfig{}, "test")

	return logger, database
}

func main() {
	logger, database := makeDb()

	finish := make(chan struct{})
	for idx := range 100 {
		go readWriteMetrics(database, logger, finish, idx)
	}

	// ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-finish:
			logger.Info().Msg("Finishing")
			return
			// case <-ticker.C:
			// 	logger.Info().Msg("Ok...")
		}
	}
}

func readWriteMetrics(db *redis.DbConnector, logger moira.Logger, finish chan<- struct{}, idx int) {
	ticker := time.NewTicker(time.Second / 10)
	i := 0

	for {
		select {
		case <-ticker.C:
			name := RandomString(10)
			i++
			// now := time.Now().Unix()
			// nowRetention := now / 60 * 60
			// value := rand.Float64()

			// err := db.SaveMetrics(map[string]*moira.MatchedMetric{
			// 	name: &moira.MatchedMetric{
			// 		Metric:             name,
			// 		Patterns:           []string{name},
			// 		Value:              value,
			// 		Timestamp:          now,
			// 		RetentionTimestamp: nowRetention,
			// 		Retention:          60,
			// 	},
			// })
			// if err != nil {
			// 	logger.Error().Error(err).Msg("Failed to save metrics")
			// 	finish <- struct{}{}
			// 	return
			// }

			// _, err = db.GetMetricsValues([]string{name}, nowRetention-60, nowRetention)
			// if err != nil {
			// 	logger.Error().Error(err).Msg("Failed to fetch metrics")
			// 	finish <- struct{}{}
			// 	return
			// }

			// _, err = db.RemoveMetricValues(name, "-inf", "inf")
			// if err != nil {
			// 	logger.Error().Error(err).Msg("Failed to remove metrics")
			// 	finish <- struct{}{}
			// 	return
			// }
			err := db.Client().Ping(context.Background()).Err()
			if err != nil {
				logger.Error().Error(err).Msg("Failed to ping")
				finish <- struct{}{}
				return
			}

			if idx == 0 && i%10 == 0 {
				logger.Info().String("name", name).Msg("Ok")
			}
		}
	}
}

func RandomString(length int) string {
	alphabet := []rune("abcdefghijklmnopqrstuvwxyz")
	var builder strings.Builder
	for _ = range length {
		builder.WriteRune(alphabet[rand.Int()%len(alphabet)])
	}
	return builder.String()
}
