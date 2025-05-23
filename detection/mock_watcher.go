// Code generated by mockery v2.52.2. DO NOT EDIT.

package detection

import mock "github.com/stretchr/testify/mock"

// MockWatcher is an autogenerated mock type for the Watcher type
type MockWatcher[T Detectable] struct {
	mock.Mock
}

type MockWatcher_Expecter[T Detectable] struct {
	mock *mock.Mock
}

func (_m *MockWatcher[T]) EXPECT() *MockWatcher_Expecter[T] {
	return &MockWatcher_Expecter[T]{mock: &_m.Mock}
}

// ErrorsChan provides a mock function with no fields
func (_m *MockWatcher[T]) ErrorsChan() chan error {
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

// MockWatcher_ErrorsChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ErrorsChan'
type MockWatcher_ErrorsChan_Call[T Detectable] struct {
	*mock.Call
}

// ErrorsChan is a helper method to define mock.On call
func (_e *MockWatcher_Expecter[T]) ErrorsChan() *MockWatcher_ErrorsChan_Call[T] {
	return &MockWatcher_ErrorsChan_Call[T]{Call: _e.mock.On("ErrorsChan")}
}

func (_c *MockWatcher_ErrorsChan_Call[T]) Run(run func()) *MockWatcher_ErrorsChan_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockWatcher_ErrorsChan_Call[T]) Return(_a0 chan error) *MockWatcher_ErrorsChan_Call[T] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockWatcher_ErrorsChan_Call[T]) RunAndReturn(run func() chan error) *MockWatcher_ErrorsChan_Call[T] {
	_c.Call.Return(run)
	return _c
}

// EventsChan provides a mock function with no fields
func (_m *MockWatcher[T]) EventsChan() chan T {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for EventsChan")
	}

	var r0 chan T
	if rf, ok := ret.Get(0).(func() chan T); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan T)
		}
	}

	return r0
}

// MockWatcher_EventsChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EventsChan'
type MockWatcher_EventsChan_Call[T Detectable] struct {
	*mock.Call
}

// EventsChan is a helper method to define mock.On call
func (_e *MockWatcher_Expecter[T]) EventsChan() *MockWatcher_EventsChan_Call[T] {
	return &MockWatcher_EventsChan_Call[T]{Call: _e.mock.On("EventsChan")}
}

func (_c *MockWatcher_EventsChan_Call[T]) Run(run func()) *MockWatcher_EventsChan_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockWatcher_EventsChan_Call[T]) Return(_a0 chan T) *MockWatcher_EventsChan_Call[T] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockWatcher_EventsChan_Call[T]) RunAndReturn(run func() chan T) *MockWatcher_EventsChan_Call[T] {
	_c.Call.Return(run)
	return _c
}

// NewMockWatcher creates a new instance of MockWatcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockWatcher[T Detectable](t interface {
	mock.TestingT
	Cleanup(func())
}) *MockWatcher[T] {
	mock := &MockWatcher[T]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
