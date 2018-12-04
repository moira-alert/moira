mockgen -package mock_moira_alert github.com/moira-alert/moira Database > ./mock/moira-alert/database.go
mockgen -package mock_moira_alert github.com/moira-alert/moira Logger > ./mock/moira-alert/logger.go
mockgen -package mock_moira_alert github.com/moira-alert/moira Sender > ./mock/moira-alert/sender.go
mockgen -package mock_notifier github.com/moira-alert/moira/notifier Notifier > ./mock/notifier/notifier.go
mockgen -package mock_scheduler github.com/moira-alert/moira/notifier Scheduler > ./mock/scheduler/scheduler.go
mockgen -package mock_moira_alert github.com/moira-alert/moira Searcher > ./mock/moira-alert/searcher.go
