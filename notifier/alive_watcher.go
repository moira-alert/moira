package notifier

import (
	"context"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type AliveWatcher struct {
	logger          moira.Logger
	database        moira.Database
	config          Config
	notifierMetrics *metrics.NotifierMetrics
}

func NewAliveWatcher(
	logger moira.Logger,
	database moira.Database,
	config Config,
	notifierMetrics *metrics.NotifierMetrics,
) *AliveWatcher {
	return &AliveWatcher{
		logger:          logger,
		database:        database,
		config:          config,
		notifierMetrics: notifierMetrics,
	}
}

func (watcher *AliveWatcher) Start(ctx context.Context) {
	go watcher.stateChecker(ctx)
}

func (watcher *AliveWatcher) stateChecker(ctx context.Context) {
	watcher.logger.Info().
		Interface("check_timeout_seconds", watcher.config.CheckNotifierStateTimeout.Seconds()).
		Msg("Moira Notifier alive watcher started")

	ticker := time.NewTicker(watcher.config.CheckNotifierStateTimeout)

	for {
		select {
		case <-ctx.Done():
			watcher.logger.Info().Msg("Moira Notifier alive watcher stopped")
			return
		case <-ticker.C:
			watcher.checkNotifierState()
		}
	}
}

func (watcher *AliveWatcher) checkNotifierState() {
	state, _ := watcher.database.GetNotifierState()
	if state != moira.SelfStateOK {
		watcher.notifierMetrics.MarkNotifierIsAlive(false)
	}

	watcher.notifierMetrics.MarkNotifierIsAlive(true)
}
