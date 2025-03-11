package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/worker"
)

const (
	deliveryCheckLockKeyPrefix = "moira-delivery-check-lock:"
	deliveryCheckLockTTL       = 30 * time.Second
	workerNameSuffix           = "DeliveryChecker"
)

func webhookLockKey(contactType string) string {
	return deliveryCheckLockKeyPrefix + contactType
}

type jsonable interface {
	json.Marshaler
	json.Unmarshaler
}

type CheckAction[T jsonable] interface {
	CheckNotificationsDelivery(fetchedDeliveryChecks []T, counter *TypesCounter) []T
}

type TypesCounter struct {
	DeliveryOK            int64
	DeliveryFailed        int64
	DeliveryChecksStopped int64
}

type Checker[T jsonable] struct {
	contactType       string
	workerName        string
	database          moira.DeliveryCheckerDatabase
	clock             moira.Clock
	logger            moira.Logger
	action            CheckAction[T]
	checkTimeout      time.Duration
	reschedulingDelay uint64
	metrics           *metrics.SenderMetrics
}

func NewChecker[T jsonable](
	contactType string,
	database moira.DeliveryCheckerDatabase,
	logger moira.Logger,
	action CheckAction[T],
	checkTimeout time.Duration,
	reschedulingDelay uint64,
	senderMetrics *metrics.SenderMetrics,
) *Checker[T] {
	workerName := contactType + " " + workerNameSuffix
	logger = logger.String("delivery.check.worker", workerName)

	return &Checker[T]{
		contactType:       contactType,
		workerName:        workerName,
		database:          database,
		clock:             clock.NewSystemClock(),
		logger:            logger,
		action:            action,
		checkTimeout:      checkTimeout,
		reschedulingDelay: reschedulingDelay,
		metrics:           senderMetrics,
	}
}

func (checker *Checker[T]) Run(stop <-chan struct{}) {
	worker.NewWorker(
		checker.workerName,
		checker.logger,
		checker.database.NewLock(webhookLockKey(checker.contactType), deliveryCheckLockTTL),
		checker.deliveryCheckerAction,
	).Run(stop)
}

func (checker *Checker[T]) deliveryCheckerAction(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(checker.checkTimeout)

	checker.logger.Info().Msg(checker.workerName + " started")
	for {
		select {
		case <-stop:
			checker.logger.Info().Msg(checker.workerName + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := checker.checkNotificationsDelivery(); err != nil {
				checker.logger.Error().
					Error(err).
					Msg("failed to perform delivery check")
			}
		}
	}
}

func (checker *Checker[T]) checkNotificationsDelivery() error {
	fetchTimestamp := checker.clock.NowUnix()

	marshaledData, err := checker.database.GetDeliveryChecksData(checker.contactType, "-inf", strconv.FormatInt(fetchTimestamp, 10))
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			// nothing to check
			return nil
		}

		return err
	}

	checksData := unmarshalChecksData[T](checker.logger, marshaledData)
	if len(checksData) == 0 {
		return nil
	}

	counter := TypesCounter{}

	checkAgainChecksData := checker.action.CheckNotificationsDelivery(checksData, &counter)

	err = checker.AddDeliveryChecks(checkAgainChecksData, checker.clock.NowUnix()+int64(checker.reschedulingDelay))
	if err != nil {
		return fmt.Errorf("failed to reschedule delivery checks: %w", err)
	}

	err = checker.removeOutdatedDeliveryChecks(fetchTimestamp)
	if err != nil {
		checker.logger.Warning().
			Error(err).
			Msg("failed to remove outdated delivery checks")
	}

	markMetrics(checker.metrics, &counter)

	return nil
}

func unmarshalChecksData[T jsonable](logger moira.Logger, marshaledData []string) []T {
	checksData := make([]T, 0, len(marshaledData))

	for _, encoded := range marshaledData {
		data := *new(T)
		err := data.UnmarshalJSON([]byte(encoded))
		if err != nil {
			logger.Warning().
				String("encoded_data", encoded).
				Error(err).
				Msg("failed to unmarshal encoded data")
			continue
		}

		checksData = append(checksData, data)
	}

	return checksData
}

func (checker *Checker[T]) AddDeliveryChecks(checksData []T, timestamp int64) error {
	if len(checksData) == 0 {
		return nil
	}

	for _, data := range checksData {
		encoded, err := data.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal check data: %w", err)
		}

		// TODO: retry operations?
		err = checker.database.AddDeliveryChecksData(checker.contactType, timestamp, string(encoded))
		if err != nil {
			return fmt.Errorf("failed to store check data: %w", err)
		}
	}

	return nil
}

func (checker *Checker[T]) removeOutdatedDeliveryChecks(lastFetchTimestamp int64) error {
	_, err := checker.database.RemoveDeliveryChecksData(checker.contactType, "-inf", strconv.FormatInt(lastFetchTimestamp, 10))
	return err
}

func markMetrics(senderMetrics *metrics.SenderMetrics, counter *TypesCounter) {
	if senderMetrics == nil || counter == nil {
		return
	}

	senderMetrics.ContactDeliveryNotificationOK.Mark(counter.DeliveryOK)
	senderMetrics.ContactDeliveryNotificationFailed.Mark(counter.DeliveryFailed)
	senderMetrics.ContactDeliveryNotificationCheckStopped.Mark(counter.DeliveryChecksStopped)
}
