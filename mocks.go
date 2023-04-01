// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go

// Package main is a generated GoMock package.
package main

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockTextScanner is a mock of TextScanner interface.
type MockTextScanner struct {
	ctrl     *gomock.Controller
	recorder *MockTextScannerMockRecorder
}

// MockTextScannerMockRecorder is the mock recorder for MockTextScanner.
type MockTextScannerMockRecorder struct {
	mock *MockTextScanner
}

// NewMockTextScanner creates a new mock instance.
func NewMockTextScanner(ctrl *gomock.Controller) *MockTextScanner {
	mock := &MockTextScanner{ctrl: ctrl}
	mock.recorder = &MockTextScannerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTextScanner) EXPECT() *MockTextScannerMockRecorder {
	return m.recorder
}

// Scan mocks base method.
func (m *MockTextScanner) Scan() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Scan")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Scan indicates an expected call of Scan.
func (mr *MockTextScannerMockRecorder) Scan() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scan", reflect.TypeOf((*MockTextScanner)(nil).Scan))
}

// Text mocks base method.
func (m *MockTextScanner) Text() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Text")
	ret0, _ := ret[0].(string)
	return ret0
}

// Text indicates an expected call of Text.
func (mr *MockTextScannerMockRecorder) Text() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Text", reflect.TypeOf((*MockTextScanner)(nil).Text))
}
