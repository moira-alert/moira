package stats

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	. "github.com/smartystreets/goconvey/convey"
)

const metricPrefix = "contacts"

var testContacts = []*moira.ContactData{
	{
		Type: "test1",
	},
	{
		Type: "test1",
	},
	{
		Type: "test2",
	},
	{
		Type: "test3",
	},
	{
		Type: "test2",
	},
	{
		Type: "test1",
	},
}

func TestNewContactsStats(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	registry := mock_metrics.NewMockRegistry(mockCtrl)
	attributedRegistry := mock_metrics.NewMockMetricRegistry(mockCtrl)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	Convey("Successfully created new contacts stats", t, func() {
		stats := NewContactStats(registry, attributedRegistry, database, logger)

		So(stats, ShouldResemble, &contactStats{
			metrics:  metrics.NewContactsMetrics(registry, attributedRegistry),
			database: database,
			logger:   logger,
		})
	})
}

func TestCheckingContactsCount(t *testing.T) {
	const contactTypeAttribute string = "contact_type"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	registry := mock_metrics.NewMockRegistry(mockCtrl)
	attributedRegistry := mock_metrics.NewMockMetricRegistry(mockCtrl)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger := mock_moira_alert.NewMockLogger(mockCtrl)
	eventBuilder := mock_moira_alert.NewMockEventBuilder(mockCtrl)

	test1Meter := mock_metrics.NewMockMeter(mockCtrl)
	test2Meter := mock_metrics.NewMockMeter(mockCtrl)
	test3Meter := mock_metrics.NewMockMeter(mockCtrl)

	var test1ContactCount, test2ContactCount, test3ContactCount int64
	var test1ContactType, test2ContactType, test3ContactType string
	test1ContactCount, test1ContactType = 3, "test1"
	test2ContactCount, test2ContactType = 2, "test2"
	test3ContactCount, test3ContactType = 1, "test3"

	getAllContactsErr := errors.New("failed to get all contacts")

	t.Run("Successfully checking contacts count", func(t *testing.T) {
		database.EXPECT().GetAllContacts().Return(testContacts, nil).Times(1)

		registry.EXPECT().NewMeter(metricPrefix, test1ContactType).Return(test1Meter).Times(1)
		registry.EXPECT().NewMeter(metricPrefix, test2ContactType).Return(test2Meter).Times(1)
		registry.EXPECT().NewMeter(metricPrefix, test3ContactType).Return(test3Meter).Times(1)
		attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
			metrics.Attribute{Key: contactTypeAttribute, Value: test1ContactType},
		}).Return(attributedRegistry)
		attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
			metrics.Attribute{Key: contactTypeAttribute, Value: test2ContactType},
		}).Return(attributedRegistry)
		attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
			metrics.Attribute{Key: contactTypeAttribute, Value: test3ContactType},
		}).Return(attributedRegistry)
		attributedRegistry.EXPECT().NewGauge(metricPrefix).Return(test1Meter, nil).Times(1)
		attributedRegistry.EXPECT().NewGauge(metricPrefix).Return(test2Meter, nil).Times(1)
		attributedRegistry.EXPECT().NewGauge(metricPrefix).Return(test3Meter, nil).Times(1)

		test1Meter.EXPECT().Mark(test1ContactCount).Times(2)
		test2Meter.EXPECT().Mark(test2ContactCount).Times(2)
		test3Meter.EXPECT().Mark(test3ContactCount).Times(2)

		stats := NewContactStats(registry, attributedRegistry, database, logger)
		stats.checkContactsCount()
		// No assertion here since all expectations are on mocks
	})

	t.Run("Get error from get all contacts", func(t *testing.T) {
		database.EXPECT().GetAllContacts().Return(nil, getAllContactsErr).Times(1)

		logger.EXPECT().Warning().Return(eventBuilder).Times(1)
		eventBuilder.EXPECT().Error(getAllContactsErr).Return(eventBuilder).Times(1)
		eventBuilder.EXPECT().Msg("Failed to get all contacts").Times(1)

		stats := NewContactStats(registry, attributedRegistry, database, logger)
		stats.checkContactsCount()
		// No assertion here since all expectations are on mocks
	})
}

