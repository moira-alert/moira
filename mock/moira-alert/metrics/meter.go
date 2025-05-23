// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/moira-alert/moira/metrics (interfaces: Meter)
//
// Generated by this command:
//
//	mockgen -destination=mock/moira-alert/metrics/meter.go -package=mock_moira_alert github.com/moira-alert/moira/metrics Meter
//

// Package mock_moira_alert is a generated GoMock package.
package mock_moira_alert

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockMeter is a mock of Meter interface.
type MockMeter struct {
	ctrl     *gomock.Controller
	recorder *MockMeterMockRecorder
	isgomock struct{}
}

// MockMeterMockRecorder is the mock recorder for MockMeter.
type MockMeterMockRecorder struct {
	mock *MockMeter
}

// NewMockMeter creates a new mock instance.
func NewMockMeter(ctrl *gomock.Controller) *MockMeter {
	mock := &MockMeter{ctrl: ctrl}
	mock.recorder = &MockMeterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMeter) EXPECT() *MockMeterMockRecorder {
	return m.recorder
}

// Count mocks base method.
func (m *MockMeter) Count() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count")
	ret0, _ := ret[0].(int64)
	return ret0
}

// Count indicates an expected call of Count.
func (mr *MockMeterMockRecorder) Count() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockMeter)(nil).Count))
}

// Mark mocks base method.
func (m *MockMeter) Mark(arg0 int64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Mark", arg0)
}

// Mark indicates an expected call of Mark.
func (mr *MockMeterMockRecorder) Mark(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mark", reflect.TypeOf((*MockMeter)(nil).Mark), arg0)
}
