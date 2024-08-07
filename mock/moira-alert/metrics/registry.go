// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/moira-alert/moira/metrics (interfaces: Registry)
//
// Generated by this command:
//
//	mockgen -destination=mock/moira-alert/metrics/registry.go -package=mock_moira_alert github.com/moira-alert/moira/metrics Registry
//

// Package mock_moira_alert is a generated GoMock package.
package mock_moira_alert

import (
	reflect "reflect"

	metrics "github.com/moira-alert/moira/metrics"
	gomock "go.uber.org/mock/gomock"
)

// MockRegistry is a mock of Registry interface.
type MockRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryMockRecorder
}

// MockRegistryMockRecorder is the mock recorder for MockRegistry.
type MockRegistryMockRecorder struct {
	mock *MockRegistry
}

// NewMockRegistry creates a new mock instance.
func NewMockRegistry(ctrl *gomock.Controller) *MockRegistry {
	mock := &MockRegistry{ctrl: ctrl}
	mock.recorder = &MockRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistry) EXPECT() *MockRegistryMockRecorder {
	return m.recorder
}

// NewCounter mocks base method.
func (m *MockRegistry) NewCounter(arg0 ...string) metrics.Counter {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewCounter", varargs...)
	ret0, _ := ret[0].(metrics.Counter)
	return ret0
}

// NewCounter indicates an expected call of NewCounter.
func (mr *MockRegistryMockRecorder) NewCounter(arg0 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewCounter", reflect.TypeOf((*MockRegistry)(nil).NewCounter), arg0...)
}

// NewHistogram mocks base method.
func (m *MockRegistry) NewHistogram(arg0 ...string) metrics.Histogram {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewHistogram", varargs...)
	ret0, _ := ret[0].(metrics.Histogram)
	return ret0
}

// NewHistogram indicates an expected call of NewHistogram.
func (mr *MockRegistryMockRecorder) NewHistogram(arg0 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewHistogram", reflect.TypeOf((*MockRegistry)(nil).NewHistogram), arg0...)
}

// NewMeter mocks base method.
func (m *MockRegistry) NewMeter(arg0 ...string) metrics.Meter {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewMeter", varargs...)
	ret0, _ := ret[0].(metrics.Meter)
	return ret0
}

// NewMeter indicates an expected call of NewMeter.
func (mr *MockRegistryMockRecorder) NewMeter(arg0 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewMeter", reflect.TypeOf((*MockRegistry)(nil).NewMeter), arg0...)
}

// NewTimer mocks base method.
func (m *MockRegistry) NewTimer(arg0 ...string) metrics.Timer {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewTimer", varargs...)
	ret0, _ := ret[0].(metrics.Timer)
	return ret0
}

// NewTimer indicates an expected call of NewTimer.
func (mr *MockRegistryMockRecorder) NewTimer(arg0 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewTimer", reflect.TypeOf((*MockRegistry)(nil).NewTimer), arg0...)
}
