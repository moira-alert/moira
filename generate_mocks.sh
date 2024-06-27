#!/bin/sh

# For windows users: Run this script via "Git bash"

rm -r ./mock/*

go install github.com/golang/mock/mockgen@v1.6.0

mockgen -destination=mock/moira-alert/locks.go -package=mock_moira_alert github.com/moira-alert/moira Lock
mockgen -destination=mock/moira-alert/mutex.go -package=mock_moira_alert github.com/moira-alert/moira Mutex
mockgen -destination=mock/moira-alert/database.go -package=mock_moira_alert github.com/moira-alert/moira Database
mockgen -destination=mock/moira-alert/image_store.go -package=mock_moira_alert github.com/moira-alert/moira ImageStore
mockgen -destination=mock/moira-alert/logger.go -package=mock_moira_alert github.com/moira-alert/moira Logger
mockgen -destination=mock/moira-alert/event_builder.go -package=mock_moira_alert github.com/moira-alert/moira/logging EventBuilder
mockgen -destination=mock/moira-alert/sender.go -package=mock_moira_alert github.com/moira-alert/moira Sender
mockgen -destination=mock/notifier/notifier.go -package=mock_notifier github.com/moira-alert/moira/notifier Notifier
mockgen -destination=mock/scheduler/scheduler.go -package=mock_scheduler github.com/moira-alert/moira/notifier Scheduler
mockgen -destination=mock/moira-alert/searcher.go -package=mock_moira_alert github.com/moira-alert/moira Searcher
mockgen -destination=mock/metric_source/source.go  -package=mock_metric_source github.com/moira-alert/moira/metric_source MetricSource
mockgen -destination=mock/metric_source/fetch_result.go -package=mock_metric_source github.com/moira-alert/moira/metric_source FetchResult
mockgen -destination=mock/heartbeat/heartbeat.go -package=mock_heartbeat github.com/moira-alert/moira/notifier/selfstate/heartbeat Heartbeater
mockgen -destination=mock/clock/clock.go -package=mock_clock github.com/moira-alert/moira Clock
mockgen -destination=mock/notifier/mattermost/client.go -package=mock_mattermost github.com/moira-alert/moira/senders/mattermost Client

mockgen -destination=mock/moira-alert/metrics/registry.go -package=mock_moira_alert github.com/moira-alert/moira/metrics Registry
mockgen -destination=mock/moira-alert/metrics/meter.go -package=mock_moira_alert github.com/moira-alert/moira/metrics Meter
mockgen -destination=mock/moira-alert/prometheus_api.go -package=mock_moira_alert github.com/moira-alert/moira/metric_source/prometheus PrometheusApi

mockgen -destination=mock/moira-alert/database_stats.go -package=mock_moira_alert github.com/moira-alert/moira/database/stats StatsReporter
mockgen -destination=mock/notifier/telegram/bot.go -package=mock_telegram github.com/moira-alert/moira/senders/telegram Bot

git add mock/*
