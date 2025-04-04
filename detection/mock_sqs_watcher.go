// Code generated by mockery v2.52.2. DO NOT EDIT.

package detection

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockSqsWatcher is an autogenerated mock type for the SqsWatcher type
type MockSqsWatcher struct {
	mock.Mock
}

type MockSqsWatcher_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSqsWatcher) EXPECT() *MockSqsWatcher_Expecter {
	return &MockSqsWatcher_Expecter{mock: &_m.Mock}
}

// ErrorsChan provides a mock function with no fields
func (_m *MockSqsWatcher) ErrorsChan() chan error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ErrorsChan")
	}

	var r0 chan error
	if rf, ok := ret.Get(0).(func() chan error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan error)
		}
	}

	return r0
}

// MockSqsWatcher_ErrorsChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ErrorsChan'
type MockSqsWatcher_ErrorsChan_Call struct {
	*mock.Call
}

// ErrorsChan is a helper method to define mock.On call
func (_e *MockSqsWatcher_Expecter) ErrorsChan() *MockSqsWatcher_ErrorsChan_Call {
	return &MockSqsWatcher_ErrorsChan_Call{Call: _e.mock.On("ErrorsChan")}
}

func (_c *MockSqsWatcher_ErrorsChan_Call) Run(run func()) *MockSqsWatcher_ErrorsChan_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSqsWatcher_ErrorsChan_Call) Return(_a0 chan error) *MockSqsWatcher_ErrorsChan_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSqsWatcher_ErrorsChan_Call) RunAndReturn(run func() chan error) *MockSqsWatcher_ErrorsChan_Call {
	_c.Call.Return(run)
	return _c
}

// EventsChan provides a mock function with no fields
func (_m *MockSqsWatcher) EventsChan() chan S3EventNotification {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for EventsChan")
	}

	var r0 chan S3EventNotification
	if rf, ok := ret.Get(0).(func() chan S3EventNotification); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan S3EventNotification)
		}
	}

	return r0
}

// MockSqsWatcher_EventsChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EventsChan'
type MockSqsWatcher_EventsChan_Call struct {
	*mock.Call
}

// EventsChan is a helper method to define mock.On call
func (_e *MockSqsWatcher_Expecter) EventsChan() *MockSqsWatcher_EventsChan_Call {
	return &MockSqsWatcher_EventsChan_Call{Call: _e.mock.On("EventsChan")}
}

func (_c *MockSqsWatcher_EventsChan_Call) Run(run func()) *MockSqsWatcher_EventsChan_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSqsWatcher_EventsChan_Call) Return(_a0 chan S3EventNotification) *MockSqsWatcher_EventsChan_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSqsWatcher_EventsChan_Call) RunAndReturn(run func() chan S3EventNotification) *MockSqsWatcher_EventsChan_Call {
	_c.Call.Return(run)
	return _c
}

// Listen provides a mock function with given fields: ctx, responder
func (_m *MockSqsWatcher) Listen(ctx context.Context, responder Responder[S3EventNotification]) {
	_m.Called(ctx, responder)
}

// MockSqsWatcher_Listen_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Listen'
type MockSqsWatcher_Listen_Call struct {
	*mock.Call
}

// Listen is a helper method to define mock.On call
//   - ctx context.Context
//   - responder Responder[S3EventNotification]
func (_e *MockSqsWatcher_Expecter) Listen(ctx interface{}, responder interface{}) *MockSqsWatcher_Listen_Call {
	return &MockSqsWatcher_Listen_Call{Call: _e.mock.On("Listen", ctx, responder)}
}

func (_c *MockSqsWatcher_Listen_Call) Run(run func(ctx context.Context, responder Responder[S3EventNotification])) *MockSqsWatcher_Listen_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(Responder[S3EventNotification]))
	})
	return _c
}

func (_c *MockSqsWatcher_Listen_Call) Return() *MockSqsWatcher_Listen_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockSqsWatcher_Listen_Call) RunAndReturn(run func(context.Context, Responder[S3EventNotification])) *MockSqsWatcher_Listen_Call {
	_c.Run(run)
	return _c
}

// StartEvents provides a mock function with given fields: ctx
func (_m *MockSqsWatcher) StartEvents(ctx context.Context) {
	_m.Called(ctx)
}

// MockSqsWatcher_StartEvents_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartEvents'
type MockSqsWatcher_StartEvents_Call struct {
	*mock.Call
}

// StartEvents is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockSqsWatcher_Expecter) StartEvents(ctx interface{}) *MockSqsWatcher_StartEvents_Call {
	return &MockSqsWatcher_StartEvents_Call{Call: _e.mock.On("StartEvents", ctx)}
}

func (_c *MockSqsWatcher_StartEvents_Call) Run(run func(ctx context.Context)) *MockSqsWatcher_StartEvents_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockSqsWatcher_StartEvents_Call) Return() *MockSqsWatcher_StartEvents_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockSqsWatcher_StartEvents_Call) RunAndReturn(run func(context.Context)) *MockSqsWatcher_StartEvents_Call {
	_c.Run(run)
	return _c
}

// NewMockSqsWatcher creates a new instance of MockSqsWatcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSqsWatcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSqsWatcher {
	mock := &MockSqsWatcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
