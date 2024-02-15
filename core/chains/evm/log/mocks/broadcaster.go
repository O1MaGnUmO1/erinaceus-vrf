// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	context "context"

	log "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/log"
	mock "github.com/stretchr/testify/mock"

	pg "github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pg"

	types "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/types"
)

// Broadcaster is an autogenerated mock type for the Broadcaster type
type Broadcaster struct {
	mock.Mock
}

// AddDependents provides a mock function with given fields: n
func (_m *Broadcaster) AddDependents(n int) {
	_m.Called(n)
}

// AwaitDependents provides a mock function with given fields:
func (_m *Broadcaster) AwaitDependents() <-chan struct{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for AwaitDependents")
	}

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *Broadcaster) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DependentReady provides a mock function with given fields:
func (_m *Broadcaster) DependentReady() {
	_m.Called()
}

// HealthReport provides a mock function with given fields:
func (_m *Broadcaster) HealthReport() map[string]error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for HealthReport")
	}

	var r0 map[string]error
	if rf, ok := ret.Get(0).(func() map[string]error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]error)
		}
	}

	return r0
}

// IsConnected provides a mock function with given fields:
func (_m *Broadcaster) IsConnected() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsConnected")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// MarkConsumed provides a mock function with given fields: lb, qopts
func (_m *Broadcaster) MarkConsumed(lb log.Broadcast, qopts ...pg.QOpt) error {
	_va := make([]interface{}, len(qopts))
	for _i := range qopts {
		_va[_i] = qopts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, lb)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for MarkConsumed")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(log.Broadcast, ...pg.QOpt) error); ok {
		r0 = rf(lb, qopts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MarkManyConsumed provides a mock function with given fields: lbs, qopts
func (_m *Broadcaster) MarkManyConsumed(lbs []log.Broadcast, qopts ...pg.QOpt) error {
	_va := make([]interface{}, len(qopts))
	for _i := range qopts {
		_va[_i] = qopts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, lbs)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for MarkManyConsumed")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]log.Broadcast, ...pg.QOpt) error); ok {
		r0 = rf(lbs, qopts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *Broadcaster) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// OnNewLongestChain provides a mock function with given fields: ctx, head
func (_m *Broadcaster) OnNewLongestChain(ctx context.Context, head *types.Head) {
	_m.Called(ctx, head)
}

// Ready provides a mock function with given fields:
func (_m *Broadcaster) Ready() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Ready")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Register provides a mock function with given fields: listener, opts
func (_m *Broadcaster) Register(listener log.Listener, opts log.ListenerOpts) func() {
	ret := _m.Called(listener, opts)

	if len(ret) == 0 {
		panic("no return value specified for Register")
	}

	var r0 func()
	if rf, ok := ret.Get(0).(func(log.Listener, log.ListenerOpts) func()); ok {
		r0 = rf(listener, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(func())
		}
	}

	return r0
}

// ReplayFromBlock provides a mock function with given fields: number, forceBroadcast
func (_m *Broadcaster) ReplayFromBlock(number int64, forceBroadcast bool) {
	_m.Called(number, forceBroadcast)
}

// Start provides a mock function with given fields: _a0
func (_m *Broadcaster) Start(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WasAlreadyConsumed provides a mock function with given fields: lb, qopts
func (_m *Broadcaster) WasAlreadyConsumed(lb log.Broadcast, qopts ...pg.QOpt) (bool, error) {
	_va := make([]interface{}, len(qopts))
	for _i := range qopts {
		_va[_i] = qopts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, lb)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for WasAlreadyConsumed")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(log.Broadcast, ...pg.QOpt) (bool, error)); ok {
		return rf(lb, qopts...)
	}
	if rf, ok := ret.Get(0).(func(log.Broadcast, ...pg.QOpt) bool); ok {
		r0 = rf(lb, qopts...)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(log.Broadcast, ...pg.QOpt) error); ok {
		r1 = rf(lb, qopts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewBroadcaster creates a new instance of Broadcaster. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBroadcaster(t interface {
	mock.TestingT
	Cleanup(func())
}) *Broadcaster {
	mock := &Broadcaster{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
