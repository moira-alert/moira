package delivery

import (
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/worker"
)

const (
	workerNameSuffix                    = "delivery checker"
	logFieldNameDeliveryCheckWorkerName = LogFieldPrefix + "worker.name"
)

type checksStorage interface {
	addManyDeliveryChecksData(timestamp int64, data []string) error
	getDeliveryChecksData(from string, to string) ([]string, error)
	removeDeliveryChecksData(from string, to string) error
}

type checksWorker struct {
	logger            moira.Logger
	clock             moira.Clock
	workerName        string
	checkTimeout      time.Duration
	reschedulingDelay uint64
	storage           checksStorage
	metrics           *metrics.SenderMetrics
	checker           NotificationDeliveryChecker
}

func newChecksWorker(
	logger moira.Logger,
	clock moira.Clock,
	workerName string,
	checkTimeout time.Duration,
	reschedulingDelay uint64,
	storage checksStorage,
	metrics *metrics.SenderMetrics,
	checker NotificationDeliveryChecker,
) *checksWorker {
	logger = logger.Clone().String(logFieldNameDeliveryCheckWorkerName, workerName)

	return &checksWorker{
		logger:            logger,
		clock:             clock,
		workerName:        workerName,
		checkTimeout:      checkTimeout,
		reschedulingDelay: reschedulingDelay,
		storage:           storage,
		metrics:           metrics,
		checker:           checker,
	}
}

func (checksWorker *checksWorker) run(lock moira.Lock, stop <-chan struct{}) {
	worker.NewWorker(
		checksWorker.workerName,
		checksWorker.logger,
		lock,
		checksWorker.deliveryCheckerAction,
	).Run(stop)
}

func (checksWorker *checksWorker) deliveryCheckerAction(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(checksWorker.checkTimeout)

	checksWorker.logger.Info().Msg(checksWorker.workerName + " started")
	for {
		select {
		case <-stop:
			checksWorker.logger.Info().Msg(checksWorker.workerName + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := checksWorker.checkNotificationsDelivery(); err != nil {
				checksWorker.logger.Error().
					Error(err).
					Msg("failed to perform delivery check")
			}
		}
	}
}

func (checksWorker *checksWorker) checkNotificationsDelivery() error {
	fetchTimestamp := checksWorker.clock.NowUnix()

	marshaledData, err := checksWorker.storage.getDeliveryChecksData("-inf", strconv.FormatInt(fetchTimestamp, 10))
	if err != nil {
		return err
	}

	if len(marshaledData) == 0 {
		return nil
	}

	checkAgainChecksData, counter := checksWorker.checker.CheckNotificationsDelivery(marshaledData)

	err = checksWorker.storage.addManyDeliveryChecksData(checksWorker.clock.NowUnix()+int64(checksWorker.reschedulingDelay), checkAgainChecksData)
	if err != nil {
		return fmt.Errorf("failed to reschedule delivery checks: %w", err)
	}

	err = checksWorker.storage.removeDeliveryChecksData("-inf", strconv.FormatInt(fetchTimestamp, 10))
	if err != nil {
		checksWorker.logger.Warning().
			Error(err).
			Msg("failed to remove outdated delivery checks")
	}

	markMetrics(checksWorker.metrics, &counter)

	return nil
}

func markMetrics(senderMetrics *metrics.SenderMetrics, counter *moira.DeliveryTypesCounter) {
	if senderMetrics == nil || counter == nil {
		return
	}

	senderMetrics.ContactDeliveryNotificationOK.Mark(counter.DeliveryOK)
	senderMetrics.ContactDeliveryNotificationFailed.Mark(counter.DeliveryFailed)
	senderMetrics.ContactDeliveryNotificationCheckStopped.Mark(counter.DeliveryChecksStopped)
}
