package delivery

import (
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics"
)

type ChecksController struct {
	database    moira.DeliveryCheckerDatabase
	lock        moira.Lock
	clock       moira.Clock
	contactType string
}

func NewChecksController(db moira.DeliveryCheckerDatabase, lock moira.Lock, contactType string) *ChecksController {
	return &ChecksController{
		database:    db,
		lock:        lock,
		clock:       clock.NewSystemClock(),
		contactType: contactType,
	}
}

func (controller *ChecksController) AddDeliveryChecksData(timestamp int64, data string) error {
	return controller.database.AddDeliveryChecksData(controller.contactType, timestamp, data)
}

func (controller *ChecksController) addManyDeliveryChecksData(timestamp int64, data []string) error {
	for _, singleData := range data {
		err := controller.AddDeliveryChecksData(timestamp, singleData)
		if err != nil {
			return fmt.Errorf("failed to store check data: %w", err)
		}
	}

	return nil
}

func (controller *ChecksController) getDeliveryChecksData(from string, to string) ([]string, error) {
	marshaledData, err := controller.database.GetDeliveryChecksData(controller.contactType, from, to)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			// nothing to check
			return nil, nil
		}

		return nil, err
	}

	return marshaledData, nil
}

func (controller *ChecksController) removeDeliveryChecksData(from string, to string) error {
	_, err := controller.database.RemoveDeliveryChecksData(controller.contactType, from, to)
	return err
}

func (controller *ChecksController) RunDeliveryChecksWorker(
	stop <-chan struct{},
	logger moira.Logger,
	checkTimeout time.Duration,
	reschedulingDelay uint64,
	metrics *metrics.SenderMetrics,
	checkAction CheckAction,
) {
	checkWorker := newChecksWorker(
		logger,
		controller.clock,
		controller.contactType+" "+workerNameSuffix,
		checkTimeout,
		reschedulingDelay,
		controller,
		metrics,
		checkAction,
	)

	go checkWorker.run(controller.lock, stop)
}
