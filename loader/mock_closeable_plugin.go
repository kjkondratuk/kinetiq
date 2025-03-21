// Code generated by mockery v2.52.2. DO NOT EDIT.

package loader

import (
	context "context"

	v1 "github.com/kjkondratuk/kinetiq/gen/kinetiq/v1"
	mock "github.com/stretchr/testify/mock"
)

// MockcloseablePlugin is an autogenerated mock type for the closeablePlugin type
type MockcloseablePlugin struct {
	mock.Mock
}

type MockcloseablePlugin_Expecter struct {
	mock *mock.Mock
}

func (_m *MockcloseablePlugin) EXPECT() *MockcloseablePlugin_Expecter {
	return &MockcloseablePlugin_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields: ctx
func (_m *MockcloseablePlugin) Close(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockcloseablePlugin_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type MockcloseablePlugin_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockcloseablePlugin_Expecter) Close(ctx interface{}) *MockcloseablePlugin_Close_Call {
	return &MockcloseablePlugin_Close_Call{Call: _e.mock.On("Close", ctx)}
}

func (_c *MockcloseablePlugin_Close_Call) Run(run func(ctx context.Context)) *MockcloseablePlugin_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockcloseablePlugin_Close_Call) Return(_a0 error) *MockcloseablePlugin_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockcloseablePlugin_Close_Call) RunAndReturn(run func(context.Context) error) *MockcloseablePlugin_Close_Call {
	_c.Call.Return(run)
	return _c
}

// Process provides a mock function with given fields: _a0, _a1
func (_m *MockcloseablePlugin) Process(_a0 context.Context, _a1 *v1.ProcessRequest) (*v1.ProcessResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Process")
	}

	var r0 *v1.ProcessResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.ProcessRequest) (*v1.ProcessResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.ProcessRequest) *v1.ProcessResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.ProcessResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.ProcessRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockcloseablePlugin_Process_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Process'
type MockcloseablePlugin_Process_Call struct {
	*mock.Call
}

// Process is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *v1.ProcessRequest
func (_e *MockcloseablePlugin_Expecter) Process(_a0 interface{}, _a1 interface{}) *MockcloseablePlugin_Process_Call {
	return &MockcloseablePlugin_Process_Call{Call: _e.mock.On("Process", _a0, _a1)}
}

func (_c *MockcloseablePlugin_Process_Call) Run(run func(_a0 context.Context, _a1 *v1.ProcessRequest)) *MockcloseablePlugin_Process_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.ProcessRequest))
	})
	return _c
}

func (_c *MockcloseablePlugin_Process_Call) Return(_a0 *v1.ProcessResponse, _a1 error) *MockcloseablePlugin_Process_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockcloseablePlugin_Process_Call) RunAndReturn(run func(context.Context, *v1.ProcessRequest) (*v1.ProcessResponse, error)) *MockcloseablePlugin_Process_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockcloseablePlugin creates a new instance of MockcloseablePlugin. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockcloseablePlugin(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockcloseablePlugin {
	mock := &MockcloseablePlugin{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
