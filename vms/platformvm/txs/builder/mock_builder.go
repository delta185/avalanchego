// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ava-labs/avalanchego/vms/platformvm/txs/builder (interfaces: Builder)

// Package builder is a generated GoMock package.
package builder

import (
	reflect "reflect"
	time "time"

	ids "github.com/ava-labs/avalanchego/ids"
	secp256k1 "github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	txs "github.com/ava-labs/avalanchego/vms/platformvm/txs"
	gomock "github.com/golang/mock/gomock"
)

// MockBuilder is a mock of Builder interface.
type MockBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockBuilderMockRecorder
}

// MockBuilderMockRecorder is the mock recorder for MockBuilder.
type MockBuilderMockRecorder struct {
	mock *MockBuilder
}

// NewMockBuilder creates a new mock instance.
func NewMockBuilder(ctrl *gomock.Controller) *MockBuilder {
	mock := &MockBuilder{ctrl: ctrl}
	mock.recorder = &MockBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBuilder) EXPECT() *MockBuilderMockRecorder {
	return m.recorder
}

// NewAddDelegatorTx mocks base method.
func (m *MockBuilder) NewAddDelegatorTx(arg0 txs.Validator, arg1 ids.ShortID, arg2 []*secp256k1.PrivateKey, arg3 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAddDelegatorTx", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewAddDelegatorTx indicates an expected call of NewAddDelegatorTx.
func (mr *MockBuilderMockRecorder) NewAddDelegatorTx(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAddDelegatorTx", reflect.TypeOf((*MockBuilder)(nil).NewAddDelegatorTx), arg0, arg1, arg2, arg3)
}

// NewAddSubnetValidatorTx mocks base method.
func (m *MockBuilder) NewAddSubnetValidatorTx(arg0, arg1, arg2 uint64, arg3 ids.NodeID, arg4 ids.ID, arg5 []*secp256k1.PrivateKey, arg6 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAddSubnetValidatorTx", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewAddSubnetValidatorTx indicates an expected call of NewAddSubnetValidatorTx.
func (mr *MockBuilderMockRecorder) NewAddSubnetValidatorTx(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAddSubnetValidatorTx", reflect.TypeOf((*MockBuilder)(nil).NewAddSubnetValidatorTx), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// NewAddValidatorTx mocks base method.
func (m *MockBuilder) NewAddValidatorTx(arg0 txs.Validator, arg1 ids.ShortID, arg2 uint32, arg3 []*secp256k1.PrivateKey, arg4 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAddValidatorTx", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewAddValidatorTx indicates an expected call of NewAddValidatorTx.
func (mr *MockBuilderMockRecorder) NewAddValidatorTx(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAddValidatorTx", reflect.TypeOf((*MockBuilder)(nil).NewAddValidatorTx), arg0, arg1, arg2, arg3, arg4)
}

// NewAdvanceTimeTx mocks base method.
func (m *MockBuilder) NewAdvanceTimeTx(arg0 time.Time) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewAdvanceTimeTx", arg0)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewAdvanceTimeTx indicates an expected call of NewAdvanceTimeTx.
func (mr *MockBuilderMockRecorder) NewAdvanceTimeTx(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAdvanceTimeTx", reflect.TypeOf((*MockBuilder)(nil).NewAdvanceTimeTx), arg0)
}

// NewCreateChainTx mocks base method.
func (m *MockBuilder) NewCreateChainTx(arg0 ids.ID, arg1 []byte, arg2 ids.ID, arg3 []ids.ID, arg4 string, arg5 []*secp256k1.PrivateKey, arg6 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewCreateChainTx", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewCreateChainTx indicates an expected call of NewCreateChainTx.
func (mr *MockBuilderMockRecorder) NewCreateChainTx(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewCreateChainTx", reflect.TypeOf((*MockBuilder)(nil).NewCreateChainTx), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// NewCreateSubnetTx mocks base method.
func (m *MockBuilder) NewCreateSubnetTx(arg0 uint32, arg1 []ids.ShortID, arg2 []*secp256k1.PrivateKey, arg3 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewCreateSubnetTx", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewCreateSubnetTx indicates an expected call of NewCreateSubnetTx.
func (mr *MockBuilderMockRecorder) NewCreateSubnetTx(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewCreateSubnetTx", reflect.TypeOf((*MockBuilder)(nil).NewCreateSubnetTx), arg0, arg1, arg2, arg3)
}

// NewExportTx mocks base method.
func (m *MockBuilder) NewExportTx(arg0 uint64, arg1 ids.ID, arg2 ids.ShortID, arg3 []*secp256k1.PrivateKey, arg4 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewExportTx", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewExportTx indicates an expected call of NewExportTx.
func (mr *MockBuilderMockRecorder) NewExportTx(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewExportTx", reflect.TypeOf((*MockBuilder)(nil).NewExportTx), arg0, arg1, arg2, arg3, arg4)
}

// NewImportTx mocks base method.
func (m *MockBuilder) NewImportTx(arg0 ids.ID, arg1 ids.ShortID, arg2 []*secp256k1.PrivateKey, arg3 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewImportTx", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewImportTx indicates an expected call of NewImportTx.
func (mr *MockBuilderMockRecorder) NewImportTx(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewImportTx", reflect.TypeOf((*MockBuilder)(nil).NewImportTx), arg0, arg1, arg2, arg3)
}

// NewRemoveSubnetValidatorTx mocks base method.
func (m *MockBuilder) NewRemoveSubnetValidatorTx(arg0 ids.NodeID, arg1 ids.ID, arg2 []*secp256k1.PrivateKey, arg3 ids.ShortID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewRemoveSubnetValidatorTx", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewRemoveSubnetValidatorTx indicates an expected call of NewRemoveSubnetValidatorTx.
func (mr *MockBuilderMockRecorder) NewRemoveSubnetValidatorTx(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewRemoveSubnetValidatorTx", reflect.TypeOf((*MockBuilder)(nil).NewRemoveSubnetValidatorTx), arg0, arg1, arg2, arg3)
}

// NewRewardValidatorTx mocks base method.
func (m *MockBuilder) NewRewardValidatorTx(arg0 ids.ID) (*txs.Tx, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewRewardValidatorTx", arg0)
	ret0, _ := ret[0].(*txs.Tx)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewRewardValidatorTx indicates an expected call of NewRewardValidatorTx.
func (mr *MockBuilderMockRecorder) NewRewardValidatorTx(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewRewardValidatorTx", reflect.TypeOf((*MockBuilder)(nil).NewRewardValidatorTx), arg0)
}
