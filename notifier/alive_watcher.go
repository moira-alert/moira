package notifier

import (
	"context"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// AliveWatcher is responsible for checking notifier state and marking notifier.alive metrics.
type AliveWatcher struct {
	logger                    moira.Logger
	database                  moira.Database
	checkNotifierStateTimeout time.Duration
	notifierMetrics           *metrics.NotifierMetrics
}

// NewAliveWatcher is an initializer for AliveWatcher.
func NewAliveWatcher(
	logger moira.Logger,
	database moira.Database,
	checkNotifierStateTimeout time.Duration,
	notifierMetrics *metrics.NotifierMetrics,
) *AliveWatcher {
	return &AliveWatcher{
		logger:                    logger,
		database:                  database,
		checkNotifierStateTimeout: checkNotifierStateTimeout,
		notifierMetrics:           notifierMetrics,
	}
}

// Start starts the checking loop in separate goroutine.
// Use context.WithCancel, context.WithTimeout etc. to terminate check loop.
func (watcher *AliveWatcher) Start(ctx context.Context) {
	go watcher.stateChecker(ctx)
}

func (watcher *AliveWatcher) stateChecker(ctx context.Context) {
	watcher.logger.Info().
		Interface("check_timeout_seconds", watcher.checkNotifierStateTimeout.Seconds()).
		Msg("Moira Notifier alive watcher started")

	ticker := time.NewTicker(watcher.checkNotifierStateTimeout)

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
	if state.NewState != moira.SelfStateOK {
		watcher.notifierMetrics.MarkNotifierIsAlive(false)
		return
	}

	watcher.notifierMetrics.MarkNotifierIsAlive(true)
}
