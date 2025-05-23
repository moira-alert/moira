// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/moira-alert/moira/logging (interfaces: EventBuilder)
//
// Generated by this command:
//
//	mockgen -destination=mock/moira-alert/event_builder.go -package=mock_moira_alert github.com/moira-alert/moira/logging EventBuilder
//

// Package mock_moira_alert is a generated GoMock package.
package mock_moira_alert

import (
	reflect "reflect"

	logging "github.com/moira-alert/moira/logging"
	gomock "go.uber.org/mock/gomock"
)

// MockEventBuilder is a mock of EventBuilder interface.
type MockEventBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockEventBuilderMockRecorder
	isgomock struct{}
}

// MockEventBuilderMockRecorder is the mock recorder for MockEventBuilder.
type MockEventBuilderMockRecorder struct {
	mock *MockEventBuilder
}

// NewMockEventBuilder creates a new mock instance.
func NewMockEventBuilder(ctrl *gomock.Controller) *MockEventBuilder {
	mock := &MockEventBuilder{ctrl: ctrl}
	mock.recorder = &MockEventBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventBuilder) EXPECT() *MockEventBuilderMockRecorder {
	return m.recorder
}

// Error mocks base method.
func (m *MockEventBuilder) Error(err error) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Error", err)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// Error indicates an expected call of Error.
func (mr *MockEventBuilderMockRecorder) Error(err any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockEventBuilder)(nil).Error), err)
}

// Fields mocks base method.
func (m *MockEventBuilder) Fields(fields map[string]any) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Fields", fields)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// Fields indicates an expected call of Fields.
func (mr *MockEventBuilderMockRecorder) Fields(fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fields", reflect.TypeOf((*MockEventBuilder)(nil).Fields), fields)
}

// Int mocks base method.
func (m *MockEventBuilder) Int(key string, value int) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Int", key, value)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// Int indicates an expected call of Int.
func (mr *MockEventBuilderMockRecorder) Int(key, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Int", reflect.TypeOf((*MockEventBuilder)(nil).Int), key, value)
}

// Int64 mocks base method.
func (m *MockEventBuilder) Int64(key string, value int64) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Int64", key, value)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// Int64 indicates an expected call of Int64.
func (mr *MockEventBuilderMockRecorder) Int64(key, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Int64", reflect.TypeOf((*MockEventBuilder)(nil).Int64), key, value)
}

// Interface mocks base method.
func (m *MockEventBuilder) Interface(key string, value any) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Interface", key, value)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// Interface indicates an expected call of Interface.
func (mr *MockEventBuilderMockRecorder) Interface(key, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Interface", reflect.TypeOf((*MockEventBuilder)(nil).Interface), key, value)
}

// Msg mocks base method.
func (m *MockEventBuilder) Msg(message string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Msg", message)
}

// Msg indicates an expected call of Msg.
func (mr *MockEventBuilderMockRecorder) Msg(message any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Msg", reflect.TypeOf((*MockEventBuilder)(nil).Msg), message)
}

// String mocks base method.
func (m *MockEventBuilder) String(key, value string) logging.EventBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String", key, value)
	ret0, _ := ret[0].(logging.EventBuilder)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockEventBuilderMockRecorder) String(key, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockEventBuilder)(nil).String), key, value)
}
