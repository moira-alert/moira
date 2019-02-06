#!/bin/sh

mockgen -destination=mock/moira-alert/locks.go -package=mock_moira_alert github.com/moira-alert/moira Lock
mockgen -destination=mock/moira-alert/database.go -package=mock_moira_alert github.com/moira-alert/moira Database
mockgen -destination=mock/moira-alert/logger.go -package=mock_moira_alert github.com/moira-alert/moira Logger
mockgen -destination=mock/moira-alert/sender.go -package=mock_moira_alert github.com/moira-alert/moira Sender
mockgen -destination=mock/notifier/notifier.go -package=mock_notifier github.com/moira-alert/moira/notifier Notifier
mockgen -destination=mock/scheduler/scheduler.go -package=mock_scheduler github.com/moira-alert/moira/notifier Scheduler
mockgen -destination=mock/moira-alert/searcher.go -package=mock_moira_alert github.com/moira-alert/moira Searcher
mockgen -destination=mock/metric_source/source.go  -package=mock_metric_source github.com/moira-alert/moira/metric_source MetricSource
mockgen -destination=mock/metric_source/fetch_result.go -package=mock_metric_source github.com/moira-alert/moira/metric_source FetchResult
