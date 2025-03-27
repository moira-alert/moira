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

// ChecksController is used to add new data to perform delivery checks and to run the checksWorker, that
// reads data from moira.DeliveryCheckerDatabase and perform checks.
type ChecksController struct {
	database    moira.DeliveryCheckerDatabase
	lock        moira.Lock
	clock       moira.Clock
	contactType string
}

// NewChecksController creates new ChecksController.
func NewChecksController(db moira.DeliveryCheckerDatabase, lock moira.Lock, contactType string) *ChecksController {
	return &ChecksController{
		database:    db,
		lock:        lock,
		clock:       clock.NewSystemClock(),
		contactType: contactType,
	}
}

// AddDeliveryChecksData schedules delivery check for given timestamp with given data.
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

// RunDeliveryChecksWorker creates and runs delivery checks worker in separate goroutine.
func (controller *ChecksController) RunDeliveryChecksWorker(
	stop <-chan struct{},
	logger moira.Logger,
	checkTimeout time.Duration,
	reschedulingDelay uint64,
	metrics *metrics.SenderMetrics,
	checker NotificationDeliveryChecker,
) {
	checkWorker := newChecksWorker(
		logger,
		controller.clock,
		makeWorkerName(controller.contactType),
		checkTimeout,
		reschedulingDelay,
		controller,
		metrics,
		checker,
	)

	go checkWorker.run(controller.lock, stop)
}

func makeWorkerName(contactType string) string {
	return contactType + " " + workerNameSuffix
}
