go get -u github.com/golang/mock/gomock
go install github.com/golang/mock/mockgen
mockgen -destination=../internal/mock/moira-alert/database.go -package=mock_moira_alert github.com/moira-alert/moira/internal/moira Database
mockgen -destination=../internal/mock/moira-alert/image_store.go -package=mock_moira_alert github.com/moira-alert/moira/internal/moira ImageStore
mockgen -destination=../internal/mock/moira-alert/logger.go -package=mock_moira_alert github.com/moira-alert/moira/internal/moira Logger
mockgen -destination=../internal/mock/moira-alert/sender.go -package=mock_moira_alert github.com/moira-alert/moira/internal/moira Sender
mockgen -destination=../internal/mock/notifier/notifier.go -package=mock_notifier github.com/moira-alert/moira/internal/notifier Notifier
mockgen -destination=../internal/mock/scheduler/scheduler.go -package=mock_scheduler github.com/moira-alert/moira/internal/notifier Scheduler
mockgen -destination=../internal/mock/moira-alert/searcher.go -package=mock_moira_alert github.com/moira-alert/moira/internal/moira Searcher
mockgen -destination=../internal/mock/metric_source/source.go  -package=mock_metric_source github.com/moira-alert/moira/internal/metric_source MetricSource
mockgen -destination=../internal/mock/metric_source/fetch_result.go -package=mock_metric_source github.com/moira-alert/moira/internal/metric_source FetchResult
