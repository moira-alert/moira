package logging

import (
	"testing"

	mocks "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDynamicLogger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockLogger := mocks.NewMockLogger(mockCtrl)
	rules := map[string]interface{}{"test_key": "test_value"}
	logger := ConfigureDynamicLog(mockLogger, rules, "debug")

	Convey("No log level changing: another field key", t, func() {
		mockLogger.EXPECT().String("another_key", "test_value")
		logger.String("another_key", "test_value")
	})

	Convey("No log level changing: another field value", t, func() {
		mockLogger.EXPECT().String("test_key", "another_value")
		logger.String("test_key", "another_value")
	})

	Convey("No log level changing: another field value type", t, func() {
		mockLogger.EXPECT().Int("test_key", 911)
		logger.Int("test_key", 911)
	})

	Convey("Change level when added field from rules", t, func() {
		mockLogger.EXPECT().Level("debug").Return(mockLogger, nil)
		mockLogger.EXPECT().String("test_key", "test_value")
		logger.String("test_key", "test_value")
	})

	Convey("Level not changed again", t, func() {
		mockLogger.EXPECT().String("test_key", "test_value")
		logger.String("test_key", "test_value")
	})
}

func TestDynamicLogger_WithSliceInRule(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockLogger := mocks.NewMockLogger(mockCtrl)
	rules := map[string]interface{}{"test_key": []int{123, 321}}
	logger := ConfigureDynamicLog(mockLogger, rules, "debug")

	Convey("No log level changing: another field key", t, func() {
		mockLogger.EXPECT().String("another_key", "test_value")
		logger.String("another_key", "test_value")
	})

	Convey("No log level changing: another field value type", t, func() {
		mockLogger.EXPECT().Int("test_key", 911)
		logger.Int("test_key", 911)
	})

	Convey("Change level when added field from rules", t, func() {
		mockLogger.EXPECT().Level("debug").Return(mockLogger, nil)
		mockLogger.EXPECT().Int("test_key", 321)
		logger.Int("test_key", 321)
	})

	Convey("Level not changed again", t, func() {
		mockLogger.EXPECT().String("test_key", "test_value")
		logger.String("test_key", "test_value")
	})
}

func TestDynamicLogger_AddFields(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockLogger := mocks.NewMockLogger(mockCtrl)
	rules := map[string]interface{}{"test_key": "test_value"}
	logger := ConfigureDynamicLog(mockLogger, rules, "debug")

	Convey("No log level changing: another field key", t, func() {
		mockLogger.EXPECT().Fields(map[string]interface{}{"another_key": "test_value"})
		logger.Fields(map[string]interface{}{"another_key": "test_value"})
	})

	Convey("No log level changing: another field value", t, func() {
		mockLogger.EXPECT().Fields(map[string]interface{}{"test_key": "another_value"})
		logger.Fields(map[string]interface{}{"test_key": "another_value"})
	})

	Convey("No log level changing: another field value type", t, func() {
		mockLogger.EXPECT().Fields(map[string]interface{}{"test_key": 911})
		logger.Fields(map[string]interface{}{"test_key": 911})
	})

	Convey("Change level when added field from rules", t, func() {
		mockLogger.EXPECT().Level("debug").Return(mockLogger, nil)
		mockLogger.EXPECT().Fields(map[string]interface{}{"test_key": "test_value"})
		logger.Fields(map[string]interface{}{"test_key": "test_value"})
	})

	Convey("Level not changed again", t, func() {
		mockLogger.EXPECT().Fields(map[string]interface{}{"test_key": "test_value"})
		logger.Fields(map[string]interface{}{"test_key": "test_value"})
	})
}
